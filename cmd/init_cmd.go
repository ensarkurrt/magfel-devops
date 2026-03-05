package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup — creates swarmforge.yml",
	Long:  `Walks you through cluster configuration and generates a swarmforge.yml file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFileExists() {
			ui.Warn("Config file already exists: %s", cfgFile)
			if !confirm("Overwrite existing config?") {
				ui.Info("Aborted")
				return nil
			}
		}

		ui.Header("SwarmForge — Interactive Setup")
		fmt.Println()

		cfg := config.DefaultConfig()
		reader := bufio.NewReader(os.Stdin)

		cfg.Cluster.Name = prompt(reader, "Cluster name", cfg.Cluster.Name)
		cfg.Hetzner.Token = prompt(reader, "Hetzner Cloud API token (or set HCLOUD_TOKEN)", cfg.Hetzner.Token)
		cfg.Hetzner.SSHKeyName = prompt(reader, "SSH key name", cfg.Hetzner.SSHKeyName)
		cfg.Hetzner.SSHKeyPath = prompt(reader, "SSH key path", cfg.Hetzner.SSHKeyPath)
		cfg.Hetzner.Image = prompt(reader, "Server image", cfg.Hetzner.Image)

		cfg.Network.Name = prompt(reader, "Network name", cfg.Network.Name)
		cfg.Network.Subnet = prompt(reader, "Subnet CIDR", cfg.Network.Subnet)

		cfg.Domains.Base = prompt(reader, "Base domain", cfg.Domains.Base)
		cfg.Domains.ACMEEmail = prompt(reader, "ACME email", cfg.Domains.ACMEEmail)

		fmt.Println()
		ui.Info("Node configuration (using defaults — edit swarmforge.yml to customize)")

		cfg.Services.PostgreSQL.User = prompt(reader, "PostgreSQL user", cfg.Services.PostgreSQL.User)
		cfg.Services.PostgreSQL.DB = prompt(reader, "PostgreSQL database", cfg.Services.PostgreSQL.DB)
		cfg.Services.MinIO.RootUser = prompt(reader, "MinIO root user", cfg.Services.MinIO.RootUser)

		cfg.Slack.WebhookURL = prompt(reader, "Slack webhook URL (optional)", "")
		cfg.Slack.Channel = prompt(reader, "Slack channel", cfg.Slack.Channel)

		// Write config
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshaling config: %w", err)
		}

		if err := os.WriteFile(cfgFile, data, 0644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		fmt.Println()
		ui.Success("Configuration saved to %s", cfgFile)
		ui.Info("Edit the file to customize node configuration, domains, and services")
		ui.Info("Then run: swarmforge up")
		return nil
	},
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func confirm(msg string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  %s (y/N): ", msg)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func init() {
	rootCmd.AddCommand(initCmd)
}
