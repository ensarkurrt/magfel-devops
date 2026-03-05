package backup

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func SyncOffsite(client *sshpkg.Client, cfg *config.Config) error {
	sb := cfg.Backup.StorageBox
	if sb.Host == "" || sb.User == "" {
		return fmt.Errorf("storage box not configured")
	}

	cmd := fmt.Sprintf(
		`rsync -avz --delete -e "ssh -p %d -o StrictHostKeyChecking=no" \
		%s/ %s@%s:%s/`,
		sb.Port, cfg.Backup.LocalDir, sb.User, sb.Host, sb.Path)

	_, err := client.Run(cmd)
	if err != nil {
		return fmt.Errorf("offsite sync failed: %w", err)
	}
	return nil
}
