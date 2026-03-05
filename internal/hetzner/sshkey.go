package hetzner

import (
	"os"
	"strings"
)

func (c *Client) CreateSSHKey(name, publicKeyPath string) error {
	pubKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	_, err = c.Run("ssh-key", "create",
		"--name", name,
		"--public-key", strings.TrimSpace(string(pubKey)),
	)
	if err != nil {
		if strings.Contains(err.Error(), "uniqueness_error") || strings.Contains(err.Error(), "already") {
			return nil
		}
		return err
	}
	return nil
}

func (c *Client) DeleteSSHKey(name string) error {
	return c.RunSilent("ssh-key", "delete", name)
}

func (c *Client) SSHKeyExists(name string) bool {
	return c.ResourceExists("ssh-key", name)
}
