package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/deploy"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/spf13/cobra"
)

var logsTail string

var logsCmd = &cobra.Command{
	Use:   "logs <service-name>",
	Short: "Show logs for a Docker service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := deploy.GetServiceLogs(client, service, logsTail)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

func init() {
	logsCmd.Flags().StringVar(&logsTail, "tail", "100", "number of lines to show")
	rootCmd.AddCommand(logsCmd)
}
