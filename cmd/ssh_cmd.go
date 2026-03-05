package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <node-name>",
	Short: "SSH into a cluster node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := mustLoadConfig()

		node := c.GetNodeByName(name)
		if node == nil {
			return fmt.Errorf("node not found: %s", name)
		}

		sshBin, err := exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("ssh not found in PATH")
		}

		sshArgs := []string{
			"ssh",
			"-i", c.Hetzner.SSHKeyPath,
			"-o", "StrictHostKeyChecking=no",
			fmt.Sprintf("root@%s", node.PrivateIP),
		}

		ui.Info("Connecting to %s (%s)...", name, node.PrivateIP)
		return syscall.Exec(sshBin, sshArgs, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
