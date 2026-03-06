package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cluster  ClusterConfig  `yaml:"cluster"`
	Hetzner  HetznerConfig  `yaml:"hetzner"`
	Network  NetworkConfig  `yaml:"network"`
	Nodes    []NodeConfig   `yaml:"nodes"`
	Domains  DomainsConfig  `yaml:"domains"`
	Services ServicesConfig `yaml:"services"`
	Netbird  NetbirdConfig  `yaml:"netbird"`
	Slack    SlackConfig    `yaml:"slack"`
	Backup   BackupConfig   `yaml:"backup"`
	Firewall FirewallConfig `yaml:"firewall"`
}

type ClusterConfig struct {
	Name string `yaml:"name"`
}

type HetznerConfig struct {
	Token      string `yaml:"token"`
	SSHKeyName string `yaml:"ssh_key_name"`
	SSHKeyPath string `yaml:"ssh_key_path"`
	Image      string `yaml:"image"`
}

type NetworkConfig struct {
	Name   string `yaml:"name"`
	Zone   string `yaml:"zone"`
	Subnet string `yaml:"subnet"`
}

type NodeConfig struct {
	Name      string            `yaml:"name"`
	Type      string            `yaml:"type"`
	Location  string            `yaml:"location"`
	PrivateIP string            `yaml:"private_ip"`
	Role      string            `yaml:"role"`
	Labels    map[string]string `yaml:"labels"`
}

type DomainsConfig struct {
	Base      string            `yaml:"base"`
	ACMEEmail string            `yaml:"acme_email"`
	Public    map[string]string `yaml:"public"`
	VPN       map[string]string `yaml:"vpn"`
	Internal  map[string]string `yaml:"internal"`
}

type ServicesConfig struct {
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	Redis      RedisConfig      `yaml:"redis"`
	MinIO      MinIOConfig      `yaml:"minio"`
	Registry   RegistryConfig   `yaml:"registry"`
	Loki       LokiConfig       `yaml:"loki"`
}

type PostgreSQLConfig struct {
	User              string `yaml:"user"`
	DB                string `yaml:"db"`
	MemoryLimit       string `yaml:"memory_limit"`
	MemoryReservation string `yaml:"memory_reservation"`
}

type RedisConfig struct {
	MaxMemory         string `yaml:"maxmemory"`
	MemoryLimit       string `yaml:"memory_limit"`
	MemoryReservation string `yaml:"memory_reservation"`
	EvictionPolicy    string `yaml:"eviction_policy"`
}

type MinIOConfig struct {
	RootUser          string `yaml:"root_user"`
	MemoryLimit       string `yaml:"memory_limit"`
	MemoryReservation string `yaml:"memory_reservation"`
}

type RegistryConfig struct {
	User string `yaml:"user"`
}

type LokiConfig struct {
	Retention string `yaml:"retention"`
}

type NetbirdConfig struct {
	Domain   string `yaml:"domain"`
	Subnet   string `yaml:"subnet"`
	SetupKey string `yaml:"setup_key"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

type BackupConfig struct {
	RetentionDays int              `yaml:"retention_days"`
	LocalDir      string           `yaml:"local_dir"`
	Schedule      string           `yaml:"schedule"`
	StorageBox    StorageBoxConfig `yaml:"storage_box"`
}

type StorageBoxConfig struct {
	User string `yaml:"user"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

type FirewallConfig struct {
	Rules []FirewallRuleConfig `yaml:"rules"`
}

type FirewallRuleConfig struct {
	Description string `yaml:"description"`
	Protocol    string `yaml:"protocol"`
	Port        string `yaml:"port,omitempty"`
	SourceIPs   []string `yaml:"source_ips"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand env vars
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Resolve token from env if empty
	if cfg.Hetzner.Token == "" {
		cfg.Hetzner.Token = os.Getenv("HCLOUD_TOKEN")
	}

	// Expand ~ in SSH key path
	if strings.HasPrefix(cfg.Hetzner.SSHKeyPath, "~") {
		home, _ := os.UserHomeDir()
		cfg.Hetzner.SSHKeyPath = filepath.Join(home, cfg.Hetzner.SSHKeyPath[1:])
	}

	return &cfg, nil
}

func (c *Config) Validate() []string {
	var errs []string

	if c.Cluster.Name == "" {
		errs = append(errs, "cluster.name is required")
	}

	if c.Hetzner.Token == "" {
		errs = append(errs, "hetzner.token is required (or set HCLOUD_TOKEN env var)")
	}
	if c.Hetzner.SSHKeyName == "" {
		errs = append(errs, "hetzner.ssh_key_name is required")
	}
	if c.Hetzner.SSHKeyPath == "" {
		errs = append(errs, "hetzner.ssh_key_path is required")
	}
	if c.Hetzner.Image == "" {
		errs = append(errs, "hetzner.image is required")
	}

	if c.Network.Name == "" {
		errs = append(errs, "network.name is required")
	}
	if c.Network.Subnet == "" {
		errs = append(errs, "network.subnet is required")
	} else {
		_, _, err := net.ParseCIDR(c.Network.Subnet)
		if err != nil {
			errs = append(errs, fmt.Sprintf("network.subnet is invalid CIDR: %s", c.Network.Subnet))
		}
	}

	if len(c.Nodes) == 0 {
		errs = append(errs, "at least one node is required")
	}

	managerCount := 0
	nodeNames := make(map[string]bool)
	nodeIPs := make(map[string]bool)
	for i, node := range c.Nodes {
		if node.Name == "" {
			errs = append(errs, fmt.Sprintf("nodes[%d].name is required", i))
		} else if nodeNames[node.Name] {
			errs = append(errs, fmt.Sprintf("duplicate node name: %s", node.Name))
		}
		nodeNames[node.Name] = true

		if node.Type == "" {
			errs = append(errs, fmt.Sprintf("nodes[%d].type is required", i))
		}
		if node.Location == "" {
			errs = append(errs, fmt.Sprintf("nodes[%d].location is required", i))
		}
		if node.PrivateIP == "" {
			errs = append(errs, fmt.Sprintf("nodes[%d].private_ip is required", i))
		} else {
			if net.ParseIP(node.PrivateIP) == nil {
				errs = append(errs, fmt.Sprintf("nodes[%d].private_ip is invalid: %s", i, node.PrivateIP))
			}
			if nodeIPs[node.PrivateIP] {
				errs = append(errs, fmt.Sprintf("duplicate node private_ip: %s", node.PrivateIP))
			}
			nodeIPs[node.PrivateIP] = true
		}
		if node.Role != "manager" && node.Role != "worker" {
			errs = append(errs, fmt.Sprintf("nodes[%d].role must be 'manager' or 'worker', got: %s", i, node.Role))
		}
		if node.Role == "manager" {
			managerCount++
		}
	}
	if managerCount == 0 {
		errs = append(errs, "at least one manager node is required")
	}

	if c.Domains.Base == "" {
		errs = append(errs, "domains.base is required")
	}
	if c.Domains.ACMEEmail == "" {
		errs = append(errs, "domains.acme_email is required")
	}

	return errs
}

func (c *Config) GetManagerNode() *NodeConfig {
	for i := range c.Nodes {
		if c.Nodes[i].Role == "manager" {
			return &c.Nodes[i]
		}
	}
	return nil
}

func (c *Config) GetWorkerNodes() []NodeConfig {
	var workers []NodeConfig
	for _, node := range c.Nodes {
		if node.Role == "worker" {
			workers = append(workers, node)
		}
	}
	return workers
}

func (c *Config) GetNodeByName(name string) *NodeConfig {
	for i := range c.Nodes {
		if c.Nodes[i].Name == name {
			return &c.Nodes[i]
		}
	}
	return nil
}

func (c *Config) GetNodeByLabel(key, value string) *NodeConfig {
	for i := range c.Nodes {
		if c.Nodes[i].Labels[key] == value {
			return &c.Nodes[i]
		}
	}
	return nil
}
