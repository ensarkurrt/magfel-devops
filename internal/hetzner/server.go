package hetzner

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ServerInfo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	PublicNet struct {
		IPv4 struct {
			IP string `json:"ip"`
		} `json:"ipv4"`
	} `json:"public_net"`
	ServerType struct {
		Name string `json:"name"`
	} `json:"server_type"`
	Datacenter struct {
		Name     string `json:"name"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
	} `json:"datacenter"`
}

func (c *Client) CreateServer(name, serverType, location, image, sshKeyName, networkName string) (string, error) {
	args := []string{
		"server", "create",
		"--name", name,
		"--type", serverType,
		"--location", location,
		"--image", image,
		"--ssh-key", sshKeyName,
	}
	_ = networkName // network attachment with specific IP is done separately in AttachToNetwork

	out, err := c.Run(args...)
	if err != nil {
		if strings.Contains(out, "uniqueness_error") || strings.Contains(out, "already") {
			return c.GetServerIP(name)
		}
		return "", err
	}

	return c.GetServerIP(name)
}

func (c *Client) GetServerIP(name string) (string, error) {
	out, err := c.RunJSON("server", "describe", name)
	if err != nil {
		return "", err
	}

	var server ServerInfo
	if err := json.Unmarshal([]byte(out), &server); err != nil {
		return "", fmt.Errorf("parsing server info: %w", err)
	}

	return server.PublicNet.IPv4.IP, nil
}

func (c *Client) DeleteServer(name string) error {
	return c.RunSilent("server", "delete", name)
}

func (c *Client) ListServers() ([]ServerInfo, error) {
	out, err := c.RunJSON("server", "list")
	if err != nil {
		return nil, err
	}
	return ParseJSONList[ServerInfo](out)
}

func (c *Client) AttachToNetwork(serverName, networkName, ip string) error {
	_, err := c.Run("server", "attach-to-network", serverName,
		"--network", networkName,
		"--ip", ip,
	)
	if err != nil && strings.Contains(err.Error(), "already") {
		return nil // idempotent
	}
	return err
}

func (c *Client) ServerExists(name string) bool {
	return c.ResourceExists("server", name)
}
