package hetzner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/ui"
)

type Client struct {
	Token   string
	Context string
}

func NewClient(token, context string) *Client {
	return &Client{
		Token:   token,
		Context: context,
	}
}

func (c *Client) Run(args ...string) (string, error) {
	fullArgs := append([]string{}, args...)
	cmd := exec.Command("hcloud", fullArgs...)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("HCLOUD_TOKEN=%s", c.Token))

	ui.Debug("hcloud %s", strings.Join(fullArgs, " "))

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("hcloud %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return output, nil
}

func (c *Client) RunJSON(args ...string) (string, error) {
	fullArgs := append(args, "-o", "json")
	return c.Run(fullArgs...)
}

func (c *Client) RunSilent(args ...string) error {
	_, err := c.Run(args...)
	return err
}

func (c *Client) ResourceExists(resource, name string) bool {
	_, err := c.Run(resource, "describe", name)
	return err == nil
}

func ParseJSONList[T any](data string) ([]T, error) {
	var items []T
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return items, nil
}

func CheckHcloudInstalled() error {
	_, err := exec.LookPath("hcloud")
	if err != nil {
		return fmt.Errorf("hcloud CLI not found. Install it: https://github.com/hetznercloud/cli")
	}
	return nil
}
