package cmd

import (
	"bufio"
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
	totalSteps := 20

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

	// Create and apply Hetzner Cloud Firewall
	ui.Step(5, totalSteps, "Creating cloud firewall")
	fwName := hc.FirewallName(c.Cluster.Name)
	if err := hc.CreateFirewall(fwName, c.Firewall.Rules); err != nil {
		return fmt.Errorf("creating firewall: %w", err)
	}
	for _, node := range c.Nodes {
		if err := hc.ApplyFirewallToServer(fwName, node.Name); err != nil {
			ui.Warn("Firewall apply for %s: %s", node.Name, err)
		}
	}
	ui.Success("Firewall %s ready (SSH, HTTP, HTTPS, NetBird)", fwName)

	// Attach to network with private IPs
	ui.Step(6, totalSteps, "Assigning private IPs")
	for _, node := range c.Nodes {
		if err := hc.AttachToNetwork(node.Name, c.Network.Name, node.PrivateIP); err != nil {
			ui.Warn("Network attach for %s: %s", node.Name, err)
		}
	}
	ui.Success("Private IPs assigned")

	// Wait for SSH
	ui.Step(7, totalSteps, "Waiting for SSH connections")
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
	ui.Step(8, totalSteps, "Setting up nodes (packages, Docker, security)")
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
		"pg_password":          "",
		"pg_repl_password":     "",
		"redis_password":       "",
		"minio_root_password":  "",
		"registry_password":    "",
		"netbird_db_password":  "",
		"netbird_relay_secret": "",
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
	if err := copyStackFiles(managerClient, c, secrets); err != nil {
		return fmt.Errorf("copying stacks: %w", err)
	}
	ui.Success("Stack files copied")

	// Deploy stacks — Phase 1: up to and including infra-netbird
	ui.Step(15, totalSteps, "Deploying infrastructure stacks")
	netbirdIdx := -1
	for i, s := range deploy.DeploymentOrder {
		if s == "infra-netbird" {
			netbirdIdx = i
			break
		}
	}
	phase1End := len(deploy.DeploymentOrder)
	if netbirdIdx >= 0 {
		phase1End = netbirdIdx + 1
	}
	for i, stackName := range deploy.DeploymentOrder[:phase1End] {
		sp := ui.NewSpinner(fmt.Sprintf("[%d/%d] Deploying %s...", i+1, len(deploy.DeploymentOrder), stackName))
		sp.Start()
		if err := deploy.DeployStack(managerClient, stackName, deploy.ComposePath(stackName)); err != nil {
			sp.StopWithError(fmt.Sprintf("Failed to deploy %s: %s", stackName, err))
			ui.Warn("Stack %s failed, continuing...", stackName)
			continue
		}
		sp.StopWithSuccess(fmt.Sprintf("Stack %s deployed", stackName))
		time.Sleep(2 * time.Second)

		// Create additional databases right after PostgreSQL is deployed and healthy.
		// NetBird and other services need their databases BEFORE they start.
		// Must run on the data node where the PostgreSQL container is running.
		if stackName == "data-postgresql" {
			dataNode := c.GetNodeByLabel("role", "data")
			if dataNode != nil && clients[dataNode.Name] != nil {
				ui.Info("Waiting for PostgreSQL to be healthy...")
				time.Sleep(15 * time.Second)
				createAdditionalDatabases(clients[dataNode.Name], secrets)
				ui.Success("Additional databases created")
			}
		}
	}

	// NetBird setup — prompt for setup key
	ui.Step(16, totalSteps, "Configuring NetBird VPN")
	managementURL := fmt.Sprintf("https://netbird.%s", c.Domains.Base)
	setupKey := c.Netbird.SetupKey
	if setupKey == "" {
		fmt.Println()
		ui.Info("NetBird management server is starting at: %s", managementURL)
		ui.Info("Waiting 20 seconds for it to come up...")
		time.Sleep(20 * time.Second)
		fmt.Println()
		ui.Info("Steps:")
		ui.Info("  1. Open %s in your browser", managementURL)
		ui.Info("  2. Create admin account (first visit)")
		ui.Info("  3. Go to Setup Keys → Create Setup Key")
		ui.Info("  4. Paste the key below")
		fmt.Println()
		fmt.Print("  NetBird Setup Key: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		setupKey = strings.TrimSpace(input)
	}
	if setupKey != "" {
		for _, node := range c.Nodes {
			if err := security.InstallNetbird(clients[node.Name], setupKey, managementURL); err != nil {
				ui.Warn("NetBird on %s: %s", node.Name, err)
			}
		}
		ui.Success("NetBird agents connected")
		// Harden SSH to VPN-only now that agents are up
		ui.Info("Hardening SSH — restricting to VPN subnet only...")
		for _, node := range c.Nodes {
			if err := security.HardenSSHtoVPN(clients[node.Name], c.Netbird.Subnet, c.Network.Subnet); err != nil {
				ui.Warn("SSH hardening on %s: %s", node.Name, err)
			}
		}
		ui.Success("SSH restricted to VPN (%s)", c.Netbird.Subnet)
	} else {
		ui.Warn("No setup key provided — NetBird agents not configured. Run 'swarmforge up' again after setup.")
	}

	// Deploy stacks — Phase 2: remaining stacks
	if netbirdIdx >= 0 && phase1End < len(deploy.DeploymentOrder) {
		ui.Step(17, totalSteps, "Deploying remaining stacks")
		for i, stackName := range deploy.DeploymentOrder[phase1End:] {
			sp := ui.NewSpinner(fmt.Sprintf("[%d/%d] Deploying %s...", phase1End+i+1, len(deploy.DeploymentOrder), stackName))
			sp.Start()
			if err := deploy.DeployStack(managerClient, stackName, deploy.ComposePath(stackName)); err != nil {
				sp.StopWithError(fmt.Sprintf("Failed to deploy %s: %s", stackName, err))
				ui.Warn("Stack %s failed, continuing...", stackName)
				continue
			}
			sp.StopWithSuccess(fmt.Sprintf("Stack %s deployed", stackName))
			time.Sleep(2 * time.Second)
		}
	}

	// Registry login on all nodes
	ui.Step(18, totalSteps, "Docker registry login on all nodes")
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

	ui.Step(19, totalSteps, "Setting up backup and offsite sync")
	dataNode := c.GetNodeByLabel("role", "data")
	if dataNode != nil {
		// Create Storage Box if not already configured
		if c.Backup.StorageBox.Host == "" {
			ui.Info("Creating Hetzner Storage Box...")
			box, _, err := hc.CreateStorageBox(
				c.Cluster.Name+"-backups",
				c.Backup.StorageBox.Type,
				c.Backup.StorageBox.Location,
				c.Hetzner.SSHKeyName,
			)
			if err != nil {
				ui.Warn("Storage Box creation failed: %s (offsite backup disabled)", err)
			} else {
				hetzner.FillStorageBoxConfig(c, box)
				ui.Success("Storage Box created: %s@%s", box.Login, box.Server)
			}
		}

		if c.Backup.StorageBox.Host != "" {
			ui.Info("Setting up Storage Box SSH key...")
			if err := backup.SetupStorageBoxSSH(clients[dataNode.Name], c); err != nil {
				ui.Warn("Storage Box SSH setup: %s", err)
			} else {
				ui.Success("Storage Box SSH configured")
			}
		}
		_ = backup.SetupCron(clients[dataNode.Name], c)
	}
	ui.Success("Backup cron configured (offsite: %v)", c.Backup.StorageBox.Host != "")

	// Health check
	ui.Step(20, totalSteps, "Running health checks")
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
		_, _ = clients[infraNode.Name].Run("mkdir -p /var/log/traefik /opt/traefik/dynamic /opt/traefik/acme /opt/configs/netbird /opt/data/portainer /opt/data/netbird")
		copyLocalFileToNodeTemplated(clients[infraNode.Name], "traefik/dynamic/routes.yml", "/opt/traefik/dynamic/routes.yml", cfg)
		copyLocalFileToNode(clients[infraNode.Name], "traefik/dynamic/middlewares.yml", "/opt/traefik/dynamic/middlewares.yml")
		copyNetbirdConfig(clients[infraNode.Name], cfg, secrets["netbird_db_password"], secrets["netbird_relay_secret"])
	}

	// Deploy redis.conf and registry htpasswd to data node
	dataNode := cfg.GetNodeByLabel("role", "data")
	if dataNode != nil {
		_, _ = clients[dataNode.Name].Run("mkdir -p /opt/configs/redis /opt/configs/registry /opt/data/redis /opt/data/postgresql /opt/data/minio /opt/data/registry")
		copyLocalFileToNode(clients[dataNode.Name], "stacks/data-redis/redis.conf", "/opt/configs/redis/redis.conf")
		// Generate registry htpasswd (bcrypt) — write password via file to avoid shell injection
		registryUser := cfg.Services.Registry.User
		registryPass := secrets["registry_password"]
		_ = clients[dataNode.Name].WriteContent("/tmp/registry-pass", registryPass)
		_, _ = clients[dataNode.Name].Run(fmt.Sprintf(
			"htpasswd -Bbin '%s' < /tmp/registry-pass > /opt/configs/registry/htpasswd && rm -f /tmp/registry-pass",
			registryUser,
		))
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

	// App node data dirs
	appNode := cfg.GetNodeByLabel("role", "app")
	if appNode != nil {
		_, _ = clients[appNode.Name].Run("mkdir -p /opt/data/gowa")
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

func copyLocalFileToNodeTemplated(client *sshpkg.Client, localPath, remotePath string, cfg *cfgpkg.Config) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		ui.Warn("Cannot read %s: %s", localPath, err)
		return
	}
	content := strings.ReplaceAll(string(data), "example.com", cfg.Domains.Base)
	if err := client.WriteContent(remotePath, content); err != nil {
		ui.Warn("Cannot write %s: %s", remotePath, err)
	}
}

func copyStackFiles(managerClient *sshpkg.Client, cfg *cfgpkg.Config, secrets map[string]string) error {
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

		// Template replacement: domain
		content := strings.ReplaceAll(string(data), "example.com", cfg.Domains.Base)

		// Stack-specific replacements
		if stackName == "infra-netbird" {
			content = strings.ReplaceAll(content, "NETBIRD_DB_PASSWORD", secrets["netbird_db_password"])
		}

		if err := managerClient.WriteContent(remoteDir+"/docker-compose.yml", content); err != nil {
			return fmt.Errorf("copying %s: %w", stackName, err)
		}

		// Copy any additional files in the stack directory
		entries, _ := os.ReadDir(localDir)
		for _, entry := range entries {
			if entry.Name() == "docker-compose.yml" {
				continue
			}
			localFile := filepath.Join(localDir, entry.Name())
			fileData, _ := os.ReadFile(localFile)
			_ = managerClient.WriteContent(filepath.Join(remoteDir, entry.Name()), string(fileData))
		}
	}
	return nil
}

// createAdditionalDatabases must run on the node where PostgreSQL is deployed (data node).
func createAdditionalDatabases(client *sshpkg.Client, secrets map[string]string) {
	dbs := []struct {
		name string
		user string
		pass string
	}{
		{"umami", "umami", swarm.GeneratePassword(24)},
		{"twenty", "twenty", swarm.GeneratePassword(24)},
		{"netbird", "netbird", secrets["netbird_db_password"]},
	}

	for _, db := range dbs {
		// CREATE DATABASE cannot run in a transaction block — run each statement separately.
		// PG 15+: public schema no longer world-writable, explicit GRANT required.
		cmd := fmt.Sprintf(`PG=$(docker ps -q -f name=data-postgresql_postgresql)
docker exec "$PG" psql -U admin -d postgres -c "SELECT 1 FROM pg_database WHERE datname='%s'" | grep -q 1 || \
  docker exec "$PG" psql -U admin -d postgres -c "CREATE DATABASE %s"
docker exec "$PG" psql -U admin -d postgres -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='%s') THEN CREATE USER %s WITH PASSWORD '%s'; END IF; END \$\$"
docker exec "$PG" psql -U admin -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE %s TO %s"
docker exec "$PG" psql -U admin -d %s -c "GRANT ALL ON SCHEMA public TO %s; ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO %s"`,
			db.name,
			db.name,
			db.user, db.user, db.pass,
			db.name, db.user,
			db.name, db.user, db.user)
		_, _ = client.Run(cmd)
	}
}

// copyNetbirdConfig reads the config.yaml template and substitutes both
// the domain placeholder and the DB password before writing to the server.
func copyNetbirdConfig(client *sshpkg.Client, cfg *cfgpkg.Config, dbPassword, relaySecret string) {
	data, err := os.ReadFile("stacks/infra-netbird/config.yaml")
	if err != nil {
		ui.Warn("Cannot read stacks/infra-netbird/config.yaml: %s", err)
		return
	}
	encryptionKey := swarm.GenerateBase64Key(32)
	content := strings.ReplaceAll(string(data), "example.com", cfg.Domains.Base)
	content = strings.ReplaceAll(content, "NETBIRD_DB_PASSWORD", dbPassword)
	content = strings.ReplaceAll(content, "NETBIRD_RELAY_SECRET", relaySecret)
	content = strings.ReplaceAll(content, `encryptionKey: ""`, `encryptionKey: "`+encryptionKey+`"`)
	if err := client.WriteContent("/opt/configs/netbird/config.yaml", content); err != nil {
		ui.Warn("Cannot write netbird config.yaml: %s", err)
	}
}

// config import needed for prepareAndCopyConfigs
var _ = strings.TrimSpace
