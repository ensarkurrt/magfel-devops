package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/security"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var firewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Manage UFW firewall on all nodes",
}

var firewallStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show UFW status on all nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		for _, node := range c.Nodes {
			client := ssh.NewClient(node.PrivateIP, "root", c.Hetzner.SSHKeyPath)
			defer client.Close()

			ui.Header(fmt.Sprintf("Node: %s (%s)", node.Name, node.PrivateIP))
			out, err := security.GetUFWStatus(client)
			if err != nil {
				ui.Error("Failed: %s", err)
				continue
			}
			fmt.Println(out)
		}
		return nil
	},
}

var firewallApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Re-apply UFW rules on all nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		for _, node := range c.Nodes {
			client := ssh.NewClient(node.PrivateIP, "root", c.Hetzner.SSHKeyPath)
			defer client.Close()

			if isDryRun() {
				ui.Info("[DRY-RUN] Would apply firewall rules on %s", node.Name)
				continue
			}

			ui.Info("Applying firewall rules on %s...", node.Name)
			if err := security.ApplyUFW(client, node, c); err != nil {
				ui.Error("Failed on %s: %s", node.Name, err)
				continue
			}
			ui.Success("Firewall rules applied on %s", node.Name)
		}
		return nil
	},
}

func init() {
	firewallCmd.AddCommand(firewallStatusCmd)
	firewallCmd.AddCommand(firewallApplyCmd)
	rootCmd.AddCommand(firewallCmd)
}
