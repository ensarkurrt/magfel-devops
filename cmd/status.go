package cmd

import (
	"fmt"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/deploy"
	"github.com/ensarkurrt/swarmforge/internal/security"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/swarm"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed cluster status report",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		clients := make(map[string]*sshpkg.Client)
		for _, node := range c.Nodes {
			client := sshpkg.NewClient(node.PrivateIP, "root", c.Hetzner.SSHKeyPath)
			if err := client.Connect(); err != nil {
				ui.Error("Cannot connect to %s (%s): %s", node.Name, node.PrivateIP, err)
				continue
			}
			defer client.Close()
			clients[node.Name] = client
		}

		// Node status
		ui.Header("Nodes")
		for _, node := range c.Nodes {
			client := clients[node.Name]
			if client == nil {
				ui.StatusLine(node.Name, "unreachable", false)
				continue
			}

			uptime := strings.TrimSpace(client.RunNoError("uptime -p"))
			cpu := strings.TrimSpace(client.RunNoError("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'.' -f1"))
			mem := strings.TrimSpace(client.RunNoError("free -m | awk '/Mem:/ {printf \"%d/%dMB (%.0f%%)\", $3, $2, $3/$2*100}'"))
			disk := strings.TrimSpace(client.RunNoError("df -h / | awk 'NR==2 {printf \"%s/%s (%s)\", $3, $2, $5}'"))
			containers := strings.TrimSpace(client.RunNoError("docker ps -q | wc -l"))

			fmt.Println()
			ui.StatusLine(fmt.Sprintf("%s (%s)", node.Name, node.Role), "online", true)
			ui.KeyValue("  Public IP", node.PrivateIP)
			ui.KeyValue("  Uptime", uptime)
			ui.KeyValue("  CPU", cpu+"%")
			ui.KeyValue("  Memory", mem)
			ui.KeyValue("  Disk", disk)
			ui.KeyValue("  Containers", containers)

			// UFW status
			ufwOut, _ := security.GetUFWStatus(client)
			ufwActive := strings.Contains(ufwOut, "active")
			ui.StatusLine("  UFW", boolStatus(ufwActive), ufwActive)

			// Fail2ban status
			f2bOut, _ := security.GetFail2banStatus(client)
			f2bActive := !strings.Contains(f2bOut, "not running")
			ui.StatusLine("  Fail2ban", boolStatus(f2bActive), f2bActive)

			// NetBird status
			nbOut := strings.TrimSpace(client.RunNoError("netbird status 2>/dev/null | head -1 || echo 'not installed'"))
			nbActive := strings.Contains(nbOut, "Connected")
			ui.StatusLine("  NetBird", nbOut, nbActive)
		}

		// Stacks
		manager := c.GetManagerNode()
		managerClient := clients[manager.Name]
		if managerClient != nil {
			ui.Header("Stacks")
			stacks, err := deploy.ListStacks(managerClient)
			if err == nil {
				fmt.Println(stacks)
			}

			// Service replicas
			deploy.CheckServiceReplicas(managerClient)

			// Health checks
			deploy.RunHealthChecks(clients)

			// Networks
			ui.Header("Overlay Networks")
			networks, err := swarm.ListNetworks(managerClient)
			if err == nil {
				fmt.Println(networks)
			}

			// Secrets
			ui.Header("Docker Secrets")
			secrets, err := swarm.ListSecrets(managerClient)
			if err == nil {
				fmt.Println(secrets)
			}

			// Swarm nodes
			ui.Header("Swarm Nodes")
			nodes, err := swarm.ListNodes(managerClient)
			if err == nil {
				fmt.Println(nodes)
			}
		}

		// Last backup
		ui.Header("Last Backup")
		for _, node := range c.Nodes {
			client := clients[node.Name]
			if client == nil {
				continue
			}
			out := strings.TrimSpace(client.RunNoError(fmt.Sprintf("ls -t %s/ 2>/dev/null | head -1", c.Backup.LocalDir)))
			if out != "" {
				ui.KeyValue("Latest backup", out)
				break
			}
		}

		return nil
	},
}

func boolStatus(ok bool) string {
	if ok {
		return "active"
	}
	return "inactive"
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
