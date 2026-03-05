package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/dns"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage internal DNS records",
}

var dnsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List internal DNS records",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := dns.ListRecords(client)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var dnsAddCmd = &cobra.Command{
	Use:   "add <hostname> <ip>",
	Short: "Add a DNS record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname, ip := args[0], args[1]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would add DNS record: %s -> %s", hostname, ip)
			return nil
		}

		if err := dns.AddRecord(client, hostname, ip); err != nil {
			return err
		}
		ui.Success("DNS record added: %s -> %s", hostname, ip)
		return nil
	},
}

var dnsRemoveCmd = &cobra.Command{
	Use:   "remove <hostname>",
	Short: "Remove a DNS record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove DNS record: %s", hostname)
			return nil
		}

		if err := dns.RemoveRecord(client, hostname); err != nil {
			return err
		}
		ui.Success("DNS record removed: %s", hostname)
		return nil
	},
}

var dnsRegenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Regenerate DNS records from config",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		hosts := dns.GenerateHostsFile(c)

		if isDryRun() {
			ui.Info("[DRY-RUN] Would regenerate DNS with:")
			fmt.Println(hosts)
			return nil
		}

		if err := dns.DeployHosts(client, hosts); err != nil {
			return err
		}
		ui.Success("DNS records regenerated from config")
		return nil
	},
}

func init() {
	dnsCmd.AddCommand(dnsListCmd)
	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsRemoveCmd)
	dnsCmd.AddCommand(dnsRegenerateCmd)
	rootCmd.AddCommand(dnsCmd)
}
