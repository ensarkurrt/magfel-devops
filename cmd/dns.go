package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/ensarkurrt/swarmforge/internal/dns"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage internal DNS records",
}

var dnsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List internal DNS records",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := dns.ListRecords(client)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var dnsAddCmd = &cobra.Command{
	Use:   "add <hostname> <ip>",
	Short: "Add a DNS record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname, ip := args[0], args[1]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would add DNS record: %s -> %s", hostname, ip)
			return nil
		}

		if err := dns.AddRecord(client, hostname, ip); err != nil {
			return err
		}
		ui.Success("DNS record added: %s -> %s", hostname, ip)
		return nil
	},
}

var dnsRemoveCmd = &cobra.Command{
	Use:   "remove <hostname>",
	Short: "Remove a DNS record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would remove DNS record: %s", hostname)
			return nil
		}

		if err := dns.RemoveRecord(client, hostname); err != nil {
			return err
		}
		ui.Success("DNS record removed: %s", hostname)
		return nil
	},
}

var dnsRegenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Regenerate DNS records from config",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		hosts := dns.GenerateHostsFile(c)

		if isDryRun() {
			ui.Info("[DRY-RUN] Would regenerate DNS with:")
			fmt.Println(hosts)
			return nil
		}

		if err := dns.DeployHosts(client, hosts); err != nil {
			return err
		}
		ui.Success("DNS records regenerated from config")
		return nil
	},
}

var dnsExportFormat string

var dnsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Generate public DNS records needed for the cluster",
	Long: `Reads domains from swarmforge.yml and outputs all DNS A records
that must be configured at your domain registrar. Supports plain text,
BIND zone file, and Hetzner DNS API formats.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		// Determine target IP
		manager := c.GetManagerNode()
		if manager == nil {
			return fmt.Errorf("no manager node found in config")
		}

		// Try to read actual public IP from .credentials
		targetIP := "<INFRA-PUBLIC-IP>"
		if data, err := os.ReadFile(".credentials"); err == nil {
			var creds map[string]interface{}
			if yaml.Unmarshal(data, &creds) == nil {
				if ips, ok := creds["server_ips"].(map[string]interface{}); ok {
					if ip, ok := ips[manager.Name].(string); ok {
						targetIP = ip
					}
				}
			}
		}

		// Collect all domains
		type dnsRecord struct {
			Domain  string
			Service string
			Access  string
		}

		var records []dnsRecord

		for name, domain := range c.Domains.Public {
			records = append(records, dnsRecord{domain, name, "public"})
		}
		for name, domain := range c.Domains.VPN {
			svcDesc := vpnServiceDescription(name)
			records = append(records, dnsRecord{domain, svcDesc, "vpn-only"})
		}

		// Sort for consistent output
		sort.Slice(records, func(i, j int) bool {
			return records[i].Domain < records[j].Domain
		})

		switch dnsExportFormat {
		case "bind":
			ui.Header(fmt.Sprintf("BIND Zone Records (%d records) — target: %s", len(records), targetIP))
			fmt.Println()
			fmt.Printf("; Public DNS records for %s cluster\n", c.Cluster.Name)
			fmt.Printf("; All records point to infra node: %s\n\n", targetIP)
			for _, r := range records {
				fmt.Printf("%-40s IN  A  %s  ; %s (%s)\n", r.Domain+".", targetIP, r.Service, r.Access)
			}

		case "table":
			ui.Header(fmt.Sprintf("DNS Records (%d records) — target: %s", len(records), targetIP))
			fmt.Println()
			fmt.Printf("%-8s %-40s %-15s %-25s %s\n", "Type", "Domain", "IP", "Service", "Access")
			fmt.Printf("%-8s %-40s %-15s %-25s %s\n", "----", "------", "--", "-------", "------")
			for _, r := range records {
				fmt.Printf("%-8s %-40s %-15s %-25s %s\n", "A", r.Domain, targetIP, r.Service, r.Access)
			}

		default: // plain
			ui.Header(fmt.Sprintf("DNS A Records (%d records)", len(records)))
			fmt.Printf("\nAll records should point to: %s\n\n", targetIP)

			fmt.Println("PUBLIC (accessible from internet):")
			for _, r := range records {
				if r.Access == "public" {
					fmt.Printf("  A  %-40s → %s    # %s\n", r.Domain, targetIP, r.Service)
				}
			}

			fmt.Println("\nVPN-ONLY (need public DNS for Let's Encrypt, access restricted by Traefik middleware):")
			for _, r := range records {
				if r.Access == "vpn-only" {
					fmt.Printf("  A  %-40s → %s    # %s\n", r.Domain, targetIP, r.Service)
				}
			}

			fmt.Printf("\nAlternative: Use wildcard record instead of %d individual records:\n", len(records))
			fmt.Printf("  A  *.%-37s → %s\n", c.Domains.Base, targetIP)
		}

		return nil
	},
}

func vpnServiceDescription(key string) string {
	descriptions := map[string]string{
		"portainer":       "Portainer CE",
		"grafana":         "Grafana",
		"minio_console":   "MinIO Console",
		"minio_api":       "MinIO S3 API",
		"traefik":         "Traefik Dashboard",
		"prometheus":      "Prometheus",
		"registry":        "Docker Registry",
		"gowa":            "GOWA WhatsApp Gateway",
		"twenty":          "Twenty CRM",
		"openstatus_dash": "OpenStatus Dashboard",
	}
	if desc, ok := descriptions[key]; ok {
		return desc
	}
	return key
}

func init() {
	dnsExportCmd.Flags().StringVar(&dnsExportFormat, "format", "plain", "output format: plain, table, bind")
	dnsCmd.AddCommand(dnsListCmd)
	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsRemoveCmd)
	dnsCmd.AddCommand(dnsRegenerateCmd)
	dnsCmd.AddCommand(dnsExportCmd)
	rootCmd.AddCommand(dnsCmd)
}
