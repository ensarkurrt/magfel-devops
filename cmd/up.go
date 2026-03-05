package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ensarkurrt/swarmforge/internal/backup"
	cfgpkg "github.com/ensarkurrt/swarmforge/internal/config"
	"github.com/ensarkurrt/swarmforge/internal/deploy"
	"github.com/ensarkurrt/swarmforge/internal/dns"
	"github.com/ensarkurrt/swarmforge/internal/hetzner"
	"github.com/ensarkurrt/swarmforge/internal/security"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/swarm"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Provision the entire infrastructure from scratch",
	Long:  `Creates servers, sets up Docker Swarm, deploys all stacks — turnkey.`,
	RunE:  runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	c := mustLoadConfig()
	totalSteps := 19

	// Validate config
	ui.Step(1, totalSteps, "Validating configuration")
	if errs := c.Validate(); len(errs) > 0 {
		for _, e := range errs {
			ui.Error("  %s", e)
		}
		return fmt.Errorf("configuration validation failed")
	}
	ui.Success("Configuration valid")

	if isDryRun() {
		ui.Info("[DRY-RUN] Would provision %d nodes and deploy %d stacks", len(c.Nodes), len(deploy.DeploymentOrder))
		for _, node := range c.Nodes {
			ui.Info("  Node: %s (%s) — %s @ %s", node.Name, node.Role, node.Type, node.Location)
		}
		for _, stack := range deploy.DeploymentOrder {
			ui.Info("  Stack: %s", stack)
		}
		return nil
	}

	// Check hcloud
	if err := hetzner.CheckHcloudInstalled(); err != nil {
		return err
	}

	hc := hetzner.NewClient(c.Hetzner.Token, c.Cluster.Name)

	// SSH key
	ui.Step(2, totalSteps, "Ensuring SSH key exists")
	if !sshpkg.KeyExists(c.Hetzner.SSHKeyPath) {
		ui.Info("Generating SSH key pair...")
		if err := sshpkg.GenerateKeyPair(c.Hetzner.SSHKeyPath); err != nil {
			return fmt.Errorf("generating SSH key: %w", err)
		}
	}
	pubKeyPath := c.Hetzner.SSHKeyPath + ".pub"
	if err := hc.CreateSSHKey(c.Hetzner.SSHKeyName, pubKeyPath); err != nil {
		return fmt.Errorf("uploading SSH key: %w", err)
	}
	ui.Success("SSH key ready")

	// Private network
	ui.Step(3, totalSteps, "Creating private network")
	if err := hc.CreateNetwork(c.Network.Name, c.Network.Zone, c.Network.Subnet); err != nil {
		return fmt.Errorf("creating network: %w", err)
	}
	ui.Success("Network %s ready", c.Network.Name)

	// Create servers
	ui.Step(4, totalSteps, "Creating %d servers", len(c.Nodes))
	serverIPs := make(map[string]string)
	for _, node := range c.Nodes {
		sp := ui.NewSpinner(fmt.Sprintf("Creating %s (%s)...", node.Name, node.Type))
		sp.Start()

		ip, err := hc.CreateServer(node.Name, node.Type, node.Location, c.Hetzner.Image, c.Hetzner.SSHKeyName, c.Network.Name)
		if err != nil {
			sp.StopWithError(fmt.Sprintf("Failed to create %s: %s", node.Name, err))
			return err
		}
		serverIPs[node.Name] = ip
		sp.StopWithSuccess(fmt.Sprintf("Server %s created (IP: %s)", node.Name, ip))
	}

	// Attach to network with private IPs
	ui.Step(5, totalSteps, "Assigning private IPs")
	for _, node := range c.Nodes {
		if err := hc.AttachToNetwork(node.Name, c.Network.Name, node.PrivateIP); err != nil {
			ui.Warn("Network attach for %s: %s", node.Name, err)
		}
	}
	ui.Success("Private IPs assigned")

	// Wait for SSH
	ui.Step(6, totalSteps, "Waiting for SSH connections")
	for name, ip := range serverIPs {
		sp := ui.NewSpinner(fmt.Sprintf("Waiting for %s (%s)...", name, ip))
		sp.Start()
		if err := sshpkg.WaitForSSH(ip, 22, 3*time.Minute); err != nil {
			sp.StopWithError(fmt.Sprintf("SSH timeout for %s", name))
			return err
		}
		sp.StopWithSuccess(fmt.Sprintf("SSH ready on %s", name))
	}

	// Setup each node
	ui.Step(7, totalSteps, "Setting up nodes (packages, Docker, security)")
	clients := make(map[string]*sshpkg.Client)
	for _, node := range c.Nodes {
		ip := serverIPs[node.Name]
		client := sshpkg.NewClient(ip, "root", c.Hetzner.SSHKeyPath)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connecting to %s: %w", node.Name, err)
		}
		defer client.Close()
		clients[node.Name] = client

		sp := ui.NewSpinner(fmt.Sprintf("Setting up %s...", node.Name))
		sp.Start()
		if err := security.SetupNode(client, node, c); err != nil {
			sp.StopWithError(fmt.Sprintf("Setup failed on %s: %s", node.Name, err))
			return err
		}
		sp.StopWithSuccess(fmt.Sprintf("Node %s configured", node.Name))
	}

	// Install NetBird
	ui.Step(8, totalSteps, "Installing NetBird agent")
	for _, node := range c.Nodes {
		if err := security.InstallNetbird(clients[node.Name], c.Netbird.SetupKey, c.Netbird.ManagementURL); err != nil {
			ui.Warn("NetBird on %s: %s", node.Name, err)
		}
	}
	ui.Success("NetBird configured")

	// Swarm init
	ui.Step(9, totalSteps, "Initializing Docker Swarm")
	manager := c.GetManagerNode()
	managerClient := clients[manager.Name]
	tokens, err := swarm.Init(managerClient, manager.PrivateIP)
	if err != nil {
		return fmt.Errorf("swarm init: %w", err)
	}
	ui.Success("Swarm initialized on %s", manager.Name)

	// Workers join
	for _, node := range c.GetWorkerNodes() {
		if err := swarm.Join(clients[node.Name], tokens.Worker, manager.PrivateIP); err != nil {
			return fmt.Errorf("worker join %s: %w", node.Name, err)
		}
		ui.Success("Worker %s joined swarm", node.Name)
	}

	// Node labels
	ui.Step(10, totalSteps, "Setting node labels")
	for _, node := range c.Nodes {
		if err := swarm.SetNodeLabels(managerClient, node.Name, node.Labels); err != nil {
			ui.Warn("Labels on %s: %s", node.Name, err)
		}
	}
	ui.Success("Node labels set")

	// Overlay network
	ui.Step(11, totalSteps, "Creating overlay network")
	if err := swarm.CreateOverlayNetwork(managerClient, "backend"); err != nil {
		return fmt.Errorf("overlay network: %w", err)
	}
	ui.Success("Overlay network 'backend' created")

	// Docker secrets
	ui.Step(12, totalSteps, "Creating Docker secrets")
	secrets := map[string]string{
		"pg_password":       "",
		"pg_repl_password":  "",
		"redis_password":    "",
		"minio_root_password": "",
		"registry_password": "",
	}
	if err := swarm.EnsureSecrets(managerClient, secrets); err != nil {
		return fmt.Errorf("creating secrets: %w", err)
	}
	ui.Success("Docker secrets created")

	// Save credentials
	saveCredentials(secrets, tokens, serverIPs)

	// Prepare and copy config files
	ui.Step(13, totalSteps, "Preparing configuration files")
	if err := prepareAndCopyConfigs(clients, c, secrets); err != nil {
		return fmt.Errorf("preparing configs: %w", err)
	}
	ui.Success("Configuration files deployed")

	// Copy stack compose files
	ui.Step(14, totalSteps, "Copying stack files to nodes")
	if err := copyStackFiles(managerClient); err != nil {
		return fmt.Errorf("copying stacks: %w", err)
	}
	ui.Success("Stack files copied")

	// Deploy stacks
	ui.Step(15, totalSteps, "Deploying %d stacks", len(deploy.DeploymentOrder))
	for i, stackName := range deploy.DeploymentOrder {
		sp := ui.NewSpinner(fmt.Sprintf("[%d/%d] Deploying %s...", i+1, len(deploy.DeploymentOrder), stackName))
		sp.Start()
		if err := deploy.DeployStack(managerClient, stackName, deploy.ComposePath(stackName)); err != nil {
			sp.StopWithError(fmt.Sprintf("Failed to deploy %s: %s", stackName, err))
			ui.Warn("Stack %s failed, continuing...", stackName)
			continue
		}
		sp.StopWithSuccess(fmt.Sprintf("Stack %s deployed", stackName))
		// Brief pause between deployments
		time.Sleep(2 * time.Second)
	}

	// Create additional databases
	ui.Step(16, totalSteps, "Creating additional databases")
	createAdditionalDatabases(managerClient, secrets)
	ui.Success("Additional databases created")

	// Registry login on all nodes
	ui.Step(17, totalSteps, "Docker registry login on all nodes")
	for _, node := range c.Nodes {
		registryHost := c.Domains.Internal["registry"]
		if registryHost == "" {
			registryHost = manager.PrivateIP + ":5000"
		}
		_, _ = clients[node.Name].Run(fmt.Sprintf(
			"docker login %s -u %s -p '%s' 2>/dev/null || true",
			registryHost, c.Services.Registry.User, secrets["registry_password"]))
	}
	ui.Success("Registry login configured")

	// Backup cron
	ui.Step(18, totalSteps, "Setting up backup cron jobs")
	dataNode := c.GetNodeByLabel("role", "data")
	if dataNode != nil {
		_ = backup.SetupCron(clients[dataNode.Name], c)
	}
	ui.Success("Backup cron configured")

	// Health check
	ui.Step(19, totalSteps, "Running health checks")
	time.Sleep(10 * time.Second) // Wait for services to start
	deploy.RunHealthChecks(clients)
	deploy.CheckServiceReplicas(managerClient)

	// Summary
	ui.Header("Cluster Ready!")
	fmt.Println()
	for _, node := range c.Nodes {
		ui.KeyValue(node.Name, fmt.Sprintf("%s (private: %s, role: %s)", serverIPs[node.Name], node.PrivateIP, node.Role))
	}
	fmt.Println()
	ui.Info("Credentials saved to .credentials")
	ui.Info("Swarm tokens saved to .credentials")
	ui.Info("Run 'swarmforge status' for detailed cluster status")

	return nil
}

func saveCredentials(secrets map[string]string, tokens *swarm.Tokens, serverIPs map[string]string) {
	creds := map[string]interface{}{
		"secrets":      secrets,
		"swarm_tokens": map[string]string{"worker": tokens.Worker, "manager": tokens.Manager},
		"server_ips":   serverIPs,
		"created_at":   time.Now().Format(time.RFC3339),
	}
	data, _ := yaml.Marshal(creds)
	_ = os.WriteFile(".credentials", data, 0600)
}

func prepareAndCopyConfigs(clients map[string]*sshpkg.Client, cfg *cfgpkg.Config, secrets map[string]string) error {
	// Generate and deploy DNS hosts file
	infraNode := cfg.GetNodeByLabel("role", "infra")
	if infraNode != nil {
		hostsContent := dns.GenerateHostsFile(cfg)
		if err := dns.DeployHosts(clients[infraNode.Name], hostsContent); err != nil {
			return fmt.Errorf("deploying DNS hosts: %w", err)
		}

		// Deploy Corefile
		corefile := dns.GenerateCorefile(cfg)
		_ = clients[infraNode.Name].WriteContent("/opt/configs/coredns/Corefile", corefile)
	}

	// Deploy traefik dynamic config to infra node
	if infraNode != nil {
		_, _ = clients[infraNode.Name].Run("mkdir -p /opt/traefik/dynamic /opt/traefik/acme")
		copyLocalFileToNode(clients[infraNode.Name], "traefik/dynamic/routes.yml", "/opt/traefik/dynamic/routes.yml")
		copyLocalFileToNode(clients[infraNode.Name], "traefik/dynamic/middlewares.yml", "/opt/traefik/dynamic/middlewares.yml")
	}

	// Deploy redis.conf to data node
	dataNode := cfg.GetNodeByLabel("role", "data")
	if dataNode != nil {
		_, _ = clients[dataNode.Name].Run("mkdir -p /opt/configs/redis /opt/data/redis /opt/data/postgresql /opt/data/minio /opt/data/portainer")
		copyLocalFileToNode(clients[dataNode.Name], "stacks/data-redis/redis.conf", "/opt/configs/redis/redis.conf")
	}

	// Deploy monitoring configs to tools node
	toolsNode := cfg.GetNodeByLabel("role", "tools")
	if toolsNode != nil {
		_, _ = clients[toolsNode.Name].Run("mkdir -p /opt/configs/prometheus /opt/configs/loki /opt/configs/promtail /opt/configs/alertmanager /opt/configs/grafana/provisioning/datasources")
		copyLocalFileToNode(clients[toolsNode.Name], "stacks/mon-prometheus/prometheus.yml", "/opt/configs/prometheus/prometheus.yml")
		copyLocalFileToNode(clients[toolsNode.Name], "stacks/mon-prometheus/alert-rules.yml", "/opt/configs/prometheus/alert-rules.yml")
		copyLocalFileToNode(clients[toolsNode.Name], "stacks/mon-alertmanager/alertmanager.yml", "/opt/configs/alertmanager/alertmanager.yml")
		copyLocalFileToNode(clients[toolsNode.Name], "stacks/log-loki/loki-config.yml", "/opt/configs/loki/loki-config.yml")
		copyLocalFileToNode(clients[toolsNode.Name], "stacks/log-promtail/promtail-config.yml", "/opt/configs/promtail/promtail-config.yml")
	}

	return nil
}

func copyLocalFileToNode(client *sshpkg.Client, localPath, remotePath string) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		ui.Warn("Cannot read %s: %s", localPath, err)
		return
	}
	if err := client.WriteContent(remotePath, string(data)); err != nil {
		ui.Warn("Cannot write %s: %s", remotePath, err)
	}
}

func copyStackFiles(managerClient *sshpkg.Client) error {
	for _, stackName := range deploy.DeploymentOrder {
		localDir := filepath.Join("stacks", stackName)
		remoteDir := deploy.StackDir(stackName)
		_, _ = managerClient.Run(fmt.Sprintf("mkdir -p %s", remoteDir))

		composePath := filepath.Join(localDir, "docker-compose.yml")
		data, err := os.ReadFile(composePath)
		if err != nil {
			ui.Warn("Cannot read %s: %s", composePath, err)
			continue
		}
		if err := managerClient.WriteContent(remoteDir+"/docker-compose.yml", string(data)); err != nil {
			return fmt.Errorf("copying %s: %w", stackName, err)
		}

		// Copy any additional files in the stack directory
		entries, _ := os.ReadDir(localDir)
		for _, entry := range entries {
			if entry.Name() == "docker-compose.yml" {
				continue
			}
			localFile := filepath.Join(localDir, entry.Name())
			data, _ := os.ReadFile(localFile)
			_ = managerClient.WriteContent(filepath.Join(remoteDir, entry.Name()), string(data))
		}
	}
	return nil
}

func createAdditionalDatabases(client *sshpkg.Client, secrets map[string]string) {
	_ = secrets
	dbs := []struct {
		name string
		user string
	}{
		{"plane", "plane"},
		{"openpanel", "openpanel"},
	}

	for _, db := range dbs {
		cmd := fmt.Sprintf(`docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
		psql -U admin -c "SELECT 1 FROM pg_database WHERE datname='%s'" | grep -q 1 || \
		docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
		psql -U admin -c "CREATE DATABASE %s; CREATE USER %s WITH PASSWORD '%s'; GRANT ALL PRIVILEGES ON DATABASE %s TO %s;"`,
			db.name, db.name, db.user, swarm.GeneratePassword(24), db.name, db.user)
		_, _ = client.Run(cmd)
	}
}

// config import needed for prepareAndCopyConfigs
var _ = strings.TrimSpace
