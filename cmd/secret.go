package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/swarm"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage Docker secrets",
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Docker secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := swarm.ListSecrets(client)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var secretSetCmd = &cobra.Command{
	Use:   "set <name> <value>",
	Short: "Create or update a Docker secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, value := args[0], args[1]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would set secret: %s", name)
			return nil
		}

		if err := swarm.UpdateSecret(client, name, value); err != nil {
			return err
		}
		ui.Success("Secret %s set", name)
		return nil
	},
}

var secretRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a Docker secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove secret: %s", name)
			return nil
		}

		if err := swarm.RemoveSecret(client, name); err != nil {
			return err
		}
		ui.Success("Secret %s removed", name)
		return nil
	},
}

func init() {
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretSetCmd)
	secretCmd.AddCommand(secretRemoveCmd)
	rootCmd.AddCommand(secretCmd)
}
