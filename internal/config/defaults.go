package config

func DefaultConfig() *Config {
	return &Config{
		Cluster: ClusterConfig{
			Name: "my-swarm",
		},
		Hetzner: HetznerConfig{
			SSHKeyName: "swarm-cluster-key",
			SSHKeyPath: "~/.ssh/id_rsa",
			Image:      "ubuntu-24.04",
		},
		Network: NetworkConfig{
			Name:   "swarm-network",
			Zone:   "eu-central",
			Subnet: "10.0.0.0/24",
		},
		Nodes: []NodeConfig{
			{Name: "swarm-infra", Type: "cx22", Location: "fsn1", PrivateIP: "10.0.0.1", Role: "manager", Labels: map[string]string{"infra": "true", "role": "infra"}},
			{Name: "swarm-data", Type: "cx32", Location: "fsn1", PrivateIP: "10.0.0.2", Role: "worker", Labels: map[string]string{"data": "true", "role": "data"}},
			{Name: "swarm-apps", Type: "cx32", Location: "fsn1", PrivateIP: "10.0.0.3", Role: "worker", Labels: map[string]string{"app": "true", "role": "app"}},
			{Name: "swarm-tools", Type: "cx32", Location: "fsn1", PrivateIP: "10.0.0.4", Role: "worker", Labels: map[string]string{"tools": "true", "role": "tools"}},
		},
		Domains: DomainsConfig{
			Base:      "example.com",
			ACMEEmail: "admin@example.com",
			Public:    map[string]string{},
			VPN:       map[string]string{},
			Internal: map[string]string{
				"pg":         "pg.internal",
				"redis":      "redis.internal",
				"minio":      "minio.internal",
				"s3":         "s3.internal",
				"portainer":  "portainer.internal",
				"traefik":    "traefik.internal",
				"dns":        "dns.internal",
				"grafana":    "grafana.internal",
				"prometheus": "prometheus.internal",
				"loki":       "loki.internal",
			},
		},
		Services: ServicesConfig{
			PostgreSQL: PostgreSQLConfig{
				User:              "admin",
				DB:                "main",
				MemoryLimit:       "2G",
				MemoryReservation: "1G",
			},
			Redis: RedisConfig{
				MaxMemory:         "512mb",
				MemoryLimit:       "1G",
				MemoryReservation: "512M",
				EvictionPolicy:    "volatile-lru",
			},
			MinIO: MinIOConfig{
				RootUser:          "minioadmin",
				MemoryLimit:       "2G",
				MemoryReservation: "1G",
			},
			Registry: RegistryConfig{
				User: "admin",
			},
			Loki: LokiConfig{
				Retention: "168h",
			},
		},
		Netbird: NetbirdConfig{
			Subnet: "100.64.0.0/10",
		},
		Slack: SlackConfig{
			Channel: "#alerts",
		},
		Backup: BackupConfig{
			RetentionDays: 14,
			LocalDir:      "/opt/backups",
			Schedule:      "0 3 * * *",
			StorageBox: StorageBoxConfig{
				Port: 23,
				Path: "/backups/swarm",
			},
		},
	}
}
