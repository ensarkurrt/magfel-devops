package security

import (
	"fmt"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func ApplyUFW(client *sshpkg.Client, node config.NodeConfig, cfg *config.Config) error {
	rules := generateUFWRules(node, cfg)

	script := `ufw --force reset
ufw default deny incoming
ufw default allow outgoing
` + strings.Join(rules, "\n") + `
ufw --force enable`

	_, err := client.Run(script)
	return err
}

func generateUFWRules(node config.NodeConfig, cfg *config.Config) []string {
	var rules []string
	privateNet := cfg.Network.Subnet
	vpnNet := cfg.Netbird.Subnet

	// Common rules for all nodes
	if vpnNet != "" {
		rules = append(rules, fmt.Sprintf("ufw allow from %s to any port 22 proto tcp comment 'SSH via VPN'", vpnNet))
	} else {
		rules = append(rules, "ufw allow 22/tcp comment 'SSH'")
	}

	// Swarm ports — private network only
	rules = append(rules,
		fmt.Sprintf("ufw allow from %s to any port 2377 proto tcp comment 'Swarm manager'", privateNet),
		fmt.Sprintf("ufw allow from %s to any port 7946 proto tcp comment 'Swarm node comm TCP'", privateNet),
		fmt.Sprintf("ufw allow from %s to any port 7946 proto udp comment 'Swarm node comm UDP'", privateNet),
		fmt.Sprintf("ufw allow from %s to any port 4789 proto udp comment 'Swarm overlay'", privateNet),
	)

	// NetBird WireGuard
	rules = append(rules, "ufw allow 51820/udp comment 'NetBird WireGuard'")

	// Role-specific rules
	role := node.Labels["role"]
	switch role {
	case "infra":
		rules = append(rules,
			"ufw allow 80/tcp comment 'HTTP'",
			"ufw allow 443/tcp comment 'HTTPS'",
			fmt.Sprintf("ufw allow from %s to any port 53 proto tcp comment 'DNS TCP'", privateNet),
			fmt.Sprintf("ufw allow from %s to any port 53 proto udp comment 'DNS UDP'", privateNet),
		)
		if vpnNet != "" {
			rules = append(rules,
				fmt.Sprintf("ufw allow from %s to any port 53 proto tcp comment 'DNS TCP VPN'", vpnNet),
				fmt.Sprintf("ufw allow from %s to any port 53 proto udp comment 'DNS UDP VPN'", vpnNet),
				fmt.Sprintf("ufw allow from %s to any port 5000 proto tcp comment 'Registry VPN'", vpnNet),
			)
		}
		rules = append(rules,
			fmt.Sprintf("ufw allow from %s to any port 5000 proto tcp comment 'Registry private'", privateNet),
		)
		// NetBird ports
		rules = append(rules,
			"ufw allow 33073/tcp comment 'NetBird management'",
			"ufw allow 10000/tcp comment 'NetBird signal'",
			"ufw allow 3478/udp comment 'NetBird TURN'",
		)

	case "data":
		rules = append(rules,
			"ufw allow 80/tcp comment 'HTTP'",
			"ufw allow 443/tcp comment 'HTTPS'",
		)
		dataPorts := []string{"5432", "6379", "9000", "9001"}
		for _, port := range dataPorts {
			rules = append(rules,
				fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Data port private'", privateNet, port),
			)
			if vpnNet != "" {
				rules = append(rules,
					fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Data port VPN'", vpnNet, port),
				)
			}
		}

	case "app":
		rules = append(rules,
			"ufw allow 80/tcp comment 'HTTP'",
			"ufw allow 443/tcp comment 'HTTPS'",
		)

	case "tools":
		rules = append(rules,
			"ufw allow 80/tcp comment 'HTTP'",
			"ufw allow 443/tcp comment 'HTTPS'",
		)
		toolPorts := []string{"3000", "3020", "3030", "9090", "3100"}
		for _, port := range toolPorts {
			if vpnNet != "" {
				rules = append(rules,
					fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Monitoring VPN'", vpnNet, port),
				)
			}
			rules = append(rules,
				fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Monitoring private'", privateNet, port),
			)
		}

	case "ci":
		rules = append(rules,
			"ufw allow 80/tcp comment 'HTTP'",
			"ufw allow 443/tcp comment 'HTTPS'",
		)
		ciPorts := []string{"8080", "50000"}
		for _, port := range ciPorts {
			if vpnNet != "" {
				rules = append(rules,
					fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Jenkins VPN'", vpnNet, port),
				)
			}
			rules = append(rules,
				fmt.Sprintf("ufw allow from %s to any port %s proto tcp comment 'Jenkins private'", privateNet, port),
			)
		}
	}

	return rules
}

func GetUFWStatus(client *sshpkg.Client) (string, error) {
	return client.Run("ufw status verbose")
}
