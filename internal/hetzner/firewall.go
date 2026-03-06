package hetzner

import (
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
)

func (c *Client) CreateFirewall(name string, rules []config.FirewallRuleConfig) error {
	_, err := c.Run("firewall", "create", "--name", name)
	if err != nil {
		if strings.Contains(err.Error(), "uniqueness_error") || strings.Contains(err.Error(), "already") {
			return nil
		}
		return err
	}

	for _, rule := range rules {
		args := []string{
			"firewall", "add-rule", name,
			"--direction", "in",
			"--protocol", rule.Protocol,
			"--description", rule.Description,
		}
		for _, ip := range rule.SourceIPs {
			args = append(args, "--source-ips", ip)
		}
		if rule.Port != "" {
			args = append(args, "--port", rule.Port)
		}
		if _, err := c.Run(args...); err != nil && !strings.Contains(err.Error(), "already") {
			return err
		}
	}

	return nil
}

func (c *Client) ApplyFirewallToServer(firewallName, serverName string) error {
	_, err := c.Run("firewall", "apply-to-resource", firewallName,
		"--type", "server",
		"--server", serverName,
	)
	if err != nil && strings.Contains(err.Error(), "already") {
		return nil
	}
	return err
}

func (c *Client) DeleteFirewall(name string) error {
	return c.RunSilent("firewall", "delete", name)
}

func (c *Client) FirewallName(clusterName string) string {
	return clusterName + "-fw"
}
