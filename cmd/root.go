package cmd

import (
	"fmt"
	"os"

	"github.com/ensarkurrt/swarmforge/internal/config"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "swarmforge",
	Short: "Turnkey Docker Swarm cluster management on Hetzner Cloud",
	Long: `SwarmForge is a professional CLI tool that provisions and manages
Docker Swarm clusters on Hetzner Cloud. It handles the full lifecycle:
infrastructure creation, Docker Swarm setup, stack deployment, networking,
security, monitoring, backup, and DNS management.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.Verbose = verbose
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%s", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "swarmforge.yml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without executing")
}

func loadConfig() (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

func mustLoadConfig() *config.Config {
	c, err := loadConfig()
	if err != nil {
		ui.Fatal("%s", err)
	}
	return c
}

func isDryRun() bool {
	return dryRun
}

func configFileExists() bool {
	_, err := os.Stat(cfgFile)
	return err == nil
}
