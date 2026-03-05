package hetzner

import "strings"

func (c *Client) CreateNetwork(name, zone, subnet string) error {
	_, err := c.Run("network", "create",
		"--name", name,
		"--ip-range", subnet,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uniqueness_error") || strings.Contains(err.Error(), "already") {
			return nil
		}
		return err
	}

	_, err = c.Run("network", "add-subnet", name,
		"--type", "cloud",
		"--network-zone", zone,
		"--ip-range", subnet,
	)
	if err != nil && strings.Contains(err.Error(), "already") {
		return nil
	}
	return err
}

func (c *Client) DeleteNetwork(name string) error {
	return c.RunSilent("network", "delete", name)
}

func (c *Client) NetworkExists(name string) bool {
	return c.ResourceExists("network", name)
}
