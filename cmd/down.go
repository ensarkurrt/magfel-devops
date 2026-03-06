package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/hetzner"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var downForce bool

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Tear down the entire infrastructure",
	Long:  `Deletes all servers, networks, and SSH keys. This action is irreversible.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		if !downForce {
			ui.Warn("This will DESTROY the entire cluster: %s", c.Cluster.Name)
			ui.Warn("All servers, data, and configurations will be permanently deleted.")
			if !confirm("Are you absolutely sure?") {
				ui.Info("Aborted")
				return nil
			}
		}

		if isDryRun() {
			ui.Info("[DRY-RUN] Would destroy:")
			for _, node := range c.Nodes {
				ui.Info("  Server: %s", node.Name)
			}
			ui.Info("  Network: %s", c.Network.Name)
			return nil
		}

		if err := hetzner.CheckHcloudInstalled(); err != nil {
			return err
		}

		hc := hetzner.NewClient(c.Hetzner.Token, c.Cluster.Name)

		// Delete firewall
		fwName := hc.FirewallName(c.Cluster.Name)
		fwSp := ui.NewSpinner(fmt.Sprintf("Deleting firewall %s...", fwName))
		fwSp.Start()
		if err := hc.DeleteFirewall(fwName); err != nil {
			fwSp.StopWithError(fmt.Sprintf("Failed to delete firewall: %s", err))
		} else {
			fwSp.StopWithSuccess("Firewall deleted")
		}

		// Delete servers
		for _, node := range c.Nodes {
			sp := ui.NewSpinner(fmt.Sprintf("Deleting server %s...", node.Name))
			sp.Start()
			if err := hc.DeleteServer(node.Name); err != nil {
				sp.StopWithError(fmt.Sprintf("Failed to delete %s: %s", node.Name, err))
			} else {
				sp.StopWithSuccess(fmt.Sprintf("Server %s deleted", node.Name))
			}
		}

		// Delete network
		sp := ui.NewSpinner("Deleting network...")
		sp.Start()
		if err := hc.DeleteNetwork(c.Network.Name); err != nil {
			sp.StopWithError(fmt.Sprintf("Failed to delete network: %s", err))
		} else {
			sp.StopWithSuccess("Network deleted")
		}

		ui.Success("Infrastructure destroyed")
		return nil
	},
}

func init() {
	downCmd.Flags().BoolVar(&downForce, "force", false, "skip confirmation prompt")
	rootCmd.AddCommand(downCmd)
}
