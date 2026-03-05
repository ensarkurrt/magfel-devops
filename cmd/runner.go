package cmd

import (
	"fmt"
	"strconv"

	cfgpkg "github.com/ensarkurrt/swarmforge/internal/config"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

const runnerStackName = "ci-runner"
const runnerServiceName = runnerStackName + "_runner"

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Manage GitHub Actions self-hosted runners",
}

var runnerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show runner service status and replica count",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		client := getRunnerClient(c)
		defer client.Close()

		ui.Header("Runner Status")
		fmt.Println()

		out, err := client.Run(fmt.Sprintf("docker service ls --filter name=%s 2>/dev/null || echo 'Runner stack not deployed'", runnerStackName))
		if err != nil {
			return err
		}
		fmt.Println(out)
		fmt.Println()

		out, err = client.Run(fmt.Sprintf(
			`docker service ps %s --format "table {{.ID}}\t{{.Name}}\t{{.CurrentState}}\t{{.Node}}\t{{.Error}}" 2>/dev/null || echo 'No tasks found'`,
			runnerServiceName))
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var runnerScaleCmd = &cobra.Command{
	Use:   "scale <count>",
	Short: "Scale runners up or down",
	Long:  `Set the number of runner replicas. Each replica handles one concurrent job.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		count, err := strconv.Atoi(args[0])
		if err != nil || count < 0 {
			return fmt.Errorf("invalid count: %s (must be a non-negative integer)", args[0])
		}

		c := mustLoadConfig()
		client := getRunnerClient(c)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would scale %s to %d replicas", runnerServiceName, count)
			return nil
		}

		ui.Info("Scaling runners to %d...", count)
		_, err = client.Run(fmt.Sprintf("docker service scale %s=%d", runnerServiceName, count))
		if err != nil {
			return fmt.Errorf("scaling runners: %w", err)
		}
		ui.Success("Runners scaled to %d replicas", count)
		return nil
	},
}

var (
	runnerDeployRepo     string
	runnerDeployPAT      string
	runnerDeployReplicas int
	runnerDeployLabels   string
)

var runnerDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the runner stack to Swarm",
	Long: `Builds the runner Docker image, pushes it to the private registry,
and deploys the ci-runner stack with the specified number of replicas.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runnerDeployRepo == "" {
			return fmt.Errorf("--repo is required")
		}
		if runnerDeployPAT == "" {
			return fmt.Errorf("--pat is required")
		}

		c := mustLoadConfig()
		client := getRunnerClient(c)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would deploy runner stack with %d replicas for %s", runnerDeployReplicas, runnerDeployRepo)
			return nil
		}

		registry := c.Domains.Internal["registry"]
		if registry == "" {
			registry = "registry:5000"
		}
		image := registry + "/github-runner:latest"

		// Build runner image on the CI node
		ui.Info("Building runner image...")
		_, err := client.Run(fmt.Sprintf(
			"docker build -t %s /opt/stacks/%s && docker push %s",
			image, runnerStackName, image))
		if err != nil {
			return fmt.Errorf("building runner image: %w", err)
		}
		ui.Success("Runner image built and pushed")

		// Deploy stack with environment variables
		ui.Info("Deploying runner stack with %d replicas...", runnerDeployReplicas)
		deployCmd := fmt.Sprintf(
			`GITHUB_REPO_URL="%s" GITHUB_PAT="%s" REGISTRY="%s" RUNNER_REPLICAS="%d" `+
				`docker stack deploy -c /opt/stacks/%s/docker-compose.yml %s`,
			runnerDeployRepo, runnerDeployPAT, registry, runnerDeployReplicas,
			runnerStackName, runnerStackName)
		_, err = client.Run(deployCmd)
		if err != nil {
			return fmt.Errorf("deploying runner stack: %w", err)
		}

		ui.Success("Runner stack deployed with %d replicas", runnerDeployReplicas)
		ui.Info("Runners will auto-register with GitHub")
		ui.Info("Scale anytime with: swarmforge runner scale <count>")
		return nil
	},
}

var runnerLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show runner service logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		client := getRunnerClient(c)
		defer client.Close()

		out, err := client.Run(fmt.Sprintf("docker service logs %s --tail 100 --no-trunc 2>&1", runnerServiceName))
		if err != nil {
			return fmt.Errorf("fetching runner logs: %w", err)
		}
		fmt.Println(out)
		return nil
	},
}

var runnerRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove the runner stack entirely",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		client := getRunnerClient(c)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove runner stack")
			return nil
		}

		ui.Info("Removing runner stack...")
		_, err := client.Run(fmt.Sprintf("docker stack rm %s", runnerStackName))
		if err != nil {
			return fmt.Errorf("removing runner stack: %w", err)
		}
		ui.Success("Runner stack removed. Runners will deregister from GitHub.")
		return nil
	},
}

func getRunnerClient(c *cfgpkg.Config) *ssh.Client {
	manager := c.GetManagerNode()
	client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
	return client
}

func init() {
	runnerDeployCmd.Flags().StringVar(&runnerDeployRepo, "repo", "", "GitHub repository URL (required)")
	runnerDeployCmd.Flags().StringVar(&runnerDeployPAT, "pat", "", "GitHub Personal Access Token (required)")
	runnerDeployCmd.Flags().IntVar(&runnerDeployReplicas, "replicas", 2, "number of runner replicas")
	runnerDeployCmd.Flags().StringVar(&runnerDeployLabels, "labels", "self-hosted,linux,swarm", "runner labels")

	runnerCmd.AddCommand(runnerStatusCmd)
	runnerCmd.AddCommand(runnerScaleCmd)
	runnerCmd.AddCommand(runnerDeployCmd)
	runnerCmd.AddCommand(runnerLogsCmd)
	runnerCmd.AddCommand(runnerRemoveCmd)
	rootCmd.AddCommand(runnerCmd)
}
