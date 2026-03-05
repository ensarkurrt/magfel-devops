package cmd

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/backup"
	"github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
	"github.com/spf13/cobra"
)

var backupOffsite bool

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
}

var backupRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a manual backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		if isDryRun() {
			ui.Info("[DRY-RUN] Would run backup (offsite: %v)", backupOffsite)
			return nil
		}

		clients := make(map[string]*ssh.Client)
		for _, node := range c.Nodes {
			client := ssh.NewClient(node.PrivateIP, "root", c.Hetzner.SSHKeyPath)
			clients[node.Name] = client
			defer client.Close()
		}

		return backup.RunBackup(clients, c, backupOffsite)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()
		manager := c.GetManagerNode()

		client := ssh.NewClient(manager.PrivateIP, "root", c.Hetzner.SSHKeyPath)
		defer client.Close()

		out, err := backup.ListBackups(client, c.Backup.LocalDir)
		if err != nil {
			return err
		}
		fmt.Println(backup.FormatBackupList(out))
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore guide and commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Header("Backup Restore Guide")
		fmt.Println()
		fmt.Println("1. PostgreSQL:")
		fmt.Println("   docker exec -i $(docker ps -q -f name=data-postgresql_postgresql) \\")
		fmt.Println("     psql -U admin < /opt/backups/<timestamp>/pg_dumpall.sql")
		fmt.Println()
		fmt.Println("2. Redis:")
		fmt.Println("   docker service scale data-redis_redis=0")
		fmt.Println("   cp /opt/backups/<timestamp>/redis-dump.rdb /opt/data/redis/dump.rdb")
		fmt.Println("   docker service scale data-redis_redis=1")
		fmt.Println()
		fmt.Println("3. MinIO:")
		fmt.Println("   tar xzf /opt/backups/<timestamp>/minio-config.tar.gz -C /opt/data/minio")
		fmt.Println()
		fmt.Println("4. Portainer:")
		fmt.Println("   docker service scale infra-portainer_portainer=0")
		fmt.Println("   tar xzf /opt/backups/<timestamp>/portainer.tar.gz -C /opt/data")
		fmt.Println("   docker service scale infra-portainer_portainer=1")
		return nil
	},
}

func init() {
	backupRunCmd.Flags().BoolVar(&backupOffsite, "offsite", false, "sync backup to offsite storage")
	backupCmd.AddCommand(backupRunCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	rootCmd.AddCommand(backupCmd)
}
