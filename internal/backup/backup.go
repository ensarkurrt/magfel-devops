package backup

import (
	"fmt"
	"strings"
	"time"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
)

func RunBackup(clients map[string]*sshpkg.Client, cfg *config.Config, offsite bool) error {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := fmt.Sprintf("%s/%s", cfg.Backup.LocalDir, timestamp)

	dataClient := getClientByRole(clients, cfg, "data")
	infraClient := getClientByRole(clients, cfg, "infra")
	toolsClient := getClientByRole(clients, cfg, "tools")

	steps := []struct {
		name   string
		client *sshpkg.Client
		fn     func(*sshpkg.Client, string) error
	}{
		{"PostgreSQL", dataClient, backupPostgreSQL},
		{"Redis", dataClient, backupRedis},
		{"MinIO config", dataClient, backupMinIO},
		{"Portainer", infraClient, backupPortainer},
		{"Traefik certs", infraClient, backupTraefik},
		{"CoreDNS config", infraClient, backupCoreDNS},
		{"Swarm state", infraClient, backupSwarmState},
		{"Stack files", toolsClient, backupStackFiles},
	}

	for _, step := range steps {
		if step.client == nil {
			ui.Warn("Skipping %s backup (no client)", step.name)
			continue
		}
		ui.Info("Backing up %s...", step.name)
		_, _ = step.client.Run(fmt.Sprintf("mkdir -p %s", backupDir))
		if err := step.fn(step.client, backupDir); err != nil {
			ui.Error("Backup %s failed: %s", step.name, err)
		} else {
			ui.Success("Backup %s complete", step.name)
		}
	}

	// Cleanup old backups
	if dataClient != nil {
		cleanupOldBackups(dataClient, cfg)
	}

	if offsite && cfg.Backup.StorageBox.Host != "" {
		return SyncOffsite(infraClient, cfg)
	}

	return nil
}

func backupPostgreSQL(client *sshpkg.Client, dir string) error {
	script := fmt.Sprintf(`docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
  pg_dumpall -U admin > %s/pg_dumpall.sql 2>/dev/null
for db in $(docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
  psql -U admin -t -c "SELECT datname FROM pg_database WHERE datistemplate = false"); do
  db=$(echo $db | xargs)
  [ -z "$db" ] && continue
  docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
    pg_dump -U admin -Fc $db > %s/pg_${db}.dump 2>/dev/null
done`, dir, dir)
	_, err := client.Run(script)
	return err
}

func backupRedis(client *sshpkg.Client, dir string) error {
	script := fmt.Sprintf(`docker exec $(docker ps -q -f name=data-redis_redis) redis-cli BGSAVE 2>/dev/null
sleep 2
cp /opt/data/redis/dump.rdb %s/redis-dump.rdb 2>/dev/null
tar czf %s/redis-aof.tar.gz -C /opt/data/redis appendonlydir 2>/dev/null || true`, dir, dir)
	_, err := client.Run(script)
	return err
}

func backupMinIO(client *sshpkg.Client, dir string) error {
	_, err := client.Run(fmt.Sprintf(
		"tar czf %s/minio-config.tar.gz -C /opt/data/minio .minio.sys 2>/dev/null || true", dir))
	return err
}

func backupPortainer(client *sshpkg.Client, dir string) error {
	_, err := client.Run(fmt.Sprintf(
		"tar czf %s/portainer.tar.gz -C /opt/data portainer 2>/dev/null || true", dir))
	return err
}

func backupTraefik(client *sshpkg.Client, dir string) error {
	_, err := client.Run(fmt.Sprintf(
		"tar czf %s/traefik.tar.gz -C /opt/traefik acme dynamic 2>/dev/null || true", dir))
	return err
}

func backupCoreDNS(client *sshpkg.Client, dir string) error {
	_, err := client.Run(fmt.Sprintf(
		"cp -r /opt/configs/coredns %s/coredns 2>/dev/null || true", dir))
	return err
}

func backupSwarmState(client *sshpkg.Client, dir string) error {
	script := fmt.Sprintf(`mkdir -p %s/swarm
docker stack ls --format '{{.Name}}' | while read stack; do
  docker stack services $stack --format json > %s/swarm/${stack}-services.json 2>/dev/null
done
cp -r /opt/stacks %s/stacks 2>/dev/null || true`, dir, dir, dir)
	_, err := client.Run(script)
	return err
}

func backupStackFiles(client *sshpkg.Client, dir string) error {
	_, err := client.Run(fmt.Sprintf(
		"cp -r /opt/stacks %s/stacks 2>/dev/null || true", dir))
	return err
}

func cleanupOldBackups(client *sshpkg.Client, cfg *config.Config) {
	_, _ = client.Run(fmt.Sprintf(
		"find %s -maxdepth 1 -type d -mtime +%d -exec rm -rf {} \\; 2>/dev/null",
		cfg.Backup.LocalDir, cfg.Backup.RetentionDays))
}

func ListBackups(client *sshpkg.Client, backupDir string) (string, error) {
	out, err := client.Run(fmt.Sprintf(
		"ls -lt %s/ 2>/dev/null | head -20 || echo 'No backups found'", backupDir))
	return out, err
}

func getClientByRole(clients map[string]*sshpkg.Client, cfg *config.Config, role string) *sshpkg.Client {
	node := cfg.GetNodeByLabel("role", role)
	if node == nil {
		return nil
	}
	return clients[node.Name]
}

func SetupCron(client *sshpkg.Client, cfg *config.Config) error {
	script := fmt.Sprintf(`cat > /opt/scripts/backup.sh << 'SCRIPT'
#!/bin/bash
TIMESTAMP=$(date +%%Y%%m%%d-%%H%%M%%S)
BACKUP_DIR="%s/$TIMESTAMP"
mkdir -p "$BACKUP_DIR"

# PostgreSQL
docker exec $(docker ps -q -f name=data-postgresql_postgresql) \
  pg_dumpall -U admin > "$BACKUP_DIR/pg_dumpall.sql" 2>/dev/null

# Redis
docker exec $(docker ps -q -f name=data-redis_redis) redis-cli BGSAVE 2>/dev/null
sleep 2
cp /opt/data/redis/dump.rdb "$BACKUP_DIR/redis-dump.rdb" 2>/dev/null

# Cleanup old
find %s -maxdepth 1 -type d -mtime +%d -exec rm -rf {} \; 2>/dev/null

echo "Backup completed: $BACKUP_DIR"
SCRIPT
chmod +x /opt/scripts/backup.sh
mkdir -p /opt/scripts

# Install cron
(crontab -l 2>/dev/null | grep -v backup.sh; echo "%s /opt/scripts/backup.sh >> /var/log/backup.log 2>&1") | crontab -
`, cfg.Backup.LocalDir, cfg.Backup.LocalDir, cfg.Backup.RetentionDays, cfg.Backup.Schedule)

	_, err := client.Run(script)
	return err
}

func FormatBackupList(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "No backups found") {
		return "No backups found"
	}
	return strings.Join(lines, "\n")
}
