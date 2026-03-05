package ssh

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ensarkurrt/swarmforge/internal/ui"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	Host       string
	User       string
	KeyPath    string
	Port       int
	conn       *ssh.Client
}

func NewClient(host, user, keyPath string) *Client {
	return &Client{
		Host:    host,
		User:    user,
		KeyPath: keyPath,
		Port:    22,
	}
}

func (c *Client) Connect() error {
	key, err := os.ReadFile(c.KeyPath)
	if err != nil {
		return fmt.Errorf("reading SSH key %s: %w", c.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("parsing SSH key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Run(command string) (string, error) {
	if c.conn == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	ui.Debug("SSH [%s]: %s", c.Host, command)

	if err := session.Run(command); err != nil {
		return stdout.String(), fmt.Errorf("command failed on %s: %w\nstderr: %s", c.Host, err, stderr.String())
	}

	return stdout.String(), nil
}

func (c *Client) RunNoError(command string) string {
	out, _ := c.Run(command)
	return out
}

func (c *Client) CopyFile(localPath, remotePath string) error {
	if c.conn == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("reading local file: %w", err)
	}

	return c.WriteFile(remotePath, data, 0644)
}

func (c *Client) WriteFile(remotePath string, data []byte, mode os.FileMode) error {
	if c.conn == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	dir := filepath.Dir(remotePath)
	base := filepath.Base(remotePath)

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%04o %d %s\n", mode, len(data), base)
		_, _ = io.Copy(w, bytes.NewReader(data))
		fmt.Fprint(w, "\x00")
	}()

	cmd := fmt.Sprintf("mkdir -p %s && scp -t %s", dir, remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("scp to %s:%s failed: %w", c.Host, remotePath, err)
	}

	return nil
}

func (c *Client) WriteContent(remotePath, content string) error {
	return c.WriteFile(remotePath, []byte(content), 0644)
}

func WaitForSSH(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("SSH connection to %s:%d timed out after %s", host, port, timeout)
}
