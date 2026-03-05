package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/deploy"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage deployed stacks",
}

var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed stacks",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := deploy.ListStacks(client)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var stackDeployCmd = &cobra.Command{
	Use:   "deploy <name>",
	Short: "Deploy a single stack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would deploy stack: %s", name)
			return nil
		}

		composePath := deploy.ComposePath(name)
		if err := deploy.DeployStack(client, name, composePath); err != nil {
			return err
		}
		ui.Success("Stack %s deployed", name)
		return nil
	},
}

var stackRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a deployed stack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove stack: %s", name)
			return nil
		}

		if err := deploy.RemoveStack(client, name); err != nil {
			return err
		}
		ui.Success("Stack %s removed", name)
		return nil
	},
}

var stackUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a stack (rolling update)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would update stack: %s", name)
			return nil
		}

		composePath := deploy.ComposePath(name)
		if err := deploy.UpdateStack(client, name, composePath); err != nil {
			return err
		}
		ui.Success("Stack %s updated", name)
		return nil
	},
}

var stackLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show logs for a stack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := deploy.GetStackLogs(client, name, false, "100")
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

func init() {
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackDeployCmd)
	stackCmd.AddCommand(stackRemoveCmd)
	stackCmd.AddCommand(stackUpdateCmd)
	stackCmd.AddCommand(stackLogsCmd)
	rootCmd.AddCommand(stackCmd)
}
