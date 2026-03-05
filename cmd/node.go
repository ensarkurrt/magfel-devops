package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/swarm"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage cluster nodes",
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes in the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()
		if manager == nil {
			return fmt.Errorf("no manager node found")
		}

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := swarm.ListNodes(client)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var nodeAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new node to the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Info("Node addition requires modifying swarmforge.yml and running 'swarmforge up'")
		ui.Info("Add the node configuration to your config file, then run 'swarmforge up'")
		return nil
	},
}

var nodeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a node from the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove node: %s", name)
			return nil
		}

		if err := swarm.RemoveNode(client, name); err != nil {
			return fmt.Errorf("removing node: %w", err)
		}
		ui.Success("Node %s removed from swarm", name)
		return nil
	},
}

var nodePromoteCmd = &cobra.Command{
	Use:   "promote <name>",
	Short: "Promote a worker to manager",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would promote node: %s", name)
			return nil
		}

		if err := swarm.PromoteNode(client, name); err != nil {
			return err
		}
		ui.Success("Node %s promoted to manager", name)
		return nil
	},
}

var nodeDemoteCmd = &cobra.Command{
	Use:   "demote <name>",
	Short: "Demote a manager to worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would demote node: %s", name)
			return nil
		}

		if err := swarm.DemoteNode(client, name); err != nil {
			return err
		}
		ui.Success("Node %s demoted to worker", name)
		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeListCmd)
	nodeCmd.AddCommand(nodeAddCmd)
	nodeCmd.AddCommand(nodeRemoveCmd)
	nodeCmd.AddCommand(nodePromoteCmd)
	nodeCmd.AddCommand(nodeDemoteCmd)
	rootCmd.AddCommand(nodeCmd)
}
