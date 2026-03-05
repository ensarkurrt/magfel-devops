package dns

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

type Record struct {
	Hostname string
	IP       string
}

func GenerateHostsFile(cfg *config.Config) string {
	var records []Record

	// Map internal domain names to their node private IPs
	internalToNode := map[string]string{
		"pg":         "data",
		"redis":      "data",
		"minio":      "data",
		"s3":         "data",
		"portainer":  "infra",
		"traefik":    "infra",
		"dns":        "infra",
		"grafana":    "tools",
		"prometheus": "tools",
		"loki":       "tools",
		"umami":      "tools",
		"twenty":     "tools",
		"openstatus": "tools",
		"backend":    "app",
		"app":        "app",
		"gowa":       "app",
	}

	for name, hostname := range cfg.Domains.Internal {
		nodeRole := internalToNode[name]
		if nodeRole == "" {
			nodeRole = "infra" // default
		}
		node := cfg.GetNodeByLabel("role", nodeRole)
		if node != nil {
			records = append(records, Record{Hostname: hostname, IP: node.PrivateIP})
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Hostname < records[j].Hostname
	})

	var lines []string
	for _, r := range records {
		lines = append(lines, fmt.Sprintf("%s %s", r.IP, r.Hostname))
	}

	return strings.Join(lines, "\n") + "\n"
}

func GenerateCorefile(cfg *config.Config) string {
	dnsNode := cfg.GetNodeByLabel("role", "infra")
	listenIP := "0.0.0.0"
	if dnsNode != nil {
		listenIP = dnsNode.PrivateIP
	}

	return fmt.Sprintf(`. {
    hosts /etc/coredns/hosts.internal {
        fallthrough
    }
    forward . 1.1.1.1 8.8.8.8
    log
    errors
    cache 30
    reload 10s
}

internal:53 {
    hosts /etc/coredns/hosts.internal {
        fallthrough
    }
    log
    errors
    cache 30
}
# Listen on %s
`, listenIP)
}

func DeployHosts(client *sshpkg.Client, hostsContent string) error {
	_, err := client.Run("mkdir -p /opt/configs/coredns")
	if err != nil {
		return err
	}
	return client.WriteContent("/opt/configs/coredns/hosts.internal", hostsContent)
}

func AddRecord(client *sshpkg.Client, hostname, ip string) error {
	entry := fmt.Sprintf("%s %s", ip, hostname)
	_, err := client.Run(fmt.Sprintf(
		`grep -q '%s' /opt/configs/coredns/hosts.internal 2>/dev/null || echo '%s' >> /opt/configs/coredns/hosts.internal`,
		hostname, entry))
	return err
}

func RemoveRecord(client *sshpkg.Client, hostname string) error {
	_, err := client.Run(fmt.Sprintf(
		`sed -i '/ %s$/d' /opt/configs/coredns/hosts.internal`, hostname))
	return err
}

func ListRecords(client *sshpkg.Client) (string, error) {
	return client.Run("cat /opt/configs/coredns/hosts.internal 2>/dev/null || echo 'No DNS records found'")
}
