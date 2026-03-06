package hetzner

import (
	"encoding/json"
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/config"
	"github.com/ensarkurrt/swarmforge/internal/swarm"
)

type StorageBoxInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Login    string `json:"login"`
	Server   string `json:"server"`
	Status   string `json:"status"`
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

// CreateStorageBox creates a Hetzner Storage Box and returns its connection info.
// If a box with the given name already exists, it returns the existing one.
func (c *Client) CreateStorageBox(name, sbType, location, sshKeyName string) (*StorageBoxInfo, string, error) {
	// Check if already exists
	existing, _ := c.RunJSON("storage-box", "list")
	if existing != "" {
		var boxes []StorageBoxInfo
		if err := json.Unmarshal([]byte(existing), &boxes); err == nil {
			for _, b := range boxes {
				if b.Name == name {
					return &b, "", nil
				}
			}
		}
	}

	password := swarm.GeneratePassword(24)

	args := []string{
		"storage-box", "create",
		"--name", name,
		"--type", sbType,
		"--location", location,
		"--password", password,
		"--enable-ssh", "true",
		"--enable-samba", "false",
		"--enable-webdav", "false",
		"-o", "json",
	}

	if sshKeyName != "" {
		args = append(args, "--ssh-key", sshKeyName)
	}

	out, err := c.Run(args...)
	if err != nil {
		return nil, "", fmt.Errorf("create storage box: %w", err)
	}

	var box StorageBoxInfo
	if err := json.Unmarshal([]byte(out), &box); err != nil {
		return nil, "", fmt.Errorf("parse storage box response: %w", err)
	}

	return &box, password, nil
}

// FillStorageBoxConfig populates the config with Storage Box connection details.
func FillStorageBoxConfig(cfg *config.Config, box *StorageBoxInfo) {
	cfg.Backup.StorageBox.User = box.Login
	cfg.Backup.StorageBox.Host = box.Server
	cfg.Backup.StorageBox.Port = 23
}
