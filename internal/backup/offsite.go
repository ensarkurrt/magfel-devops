package backup

import (
	"fmt"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
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

// SetupStorageBoxSSH installs the node's SSH public key on the Hetzner Storage Box
// so rsync can authenticate without a password.
func SetupStorageBoxSSH(client *sshpkg.Client, cfg *config.Config) error {
	sb := cfg.Backup.StorageBox
	if sb.Host == "" || sb.User == "" {
		ui.Warn("Storage Box not configured, skipping SSH key setup")
		return nil
	}

	script := fmt.Sprintf(`
# Generate SSH key if not exists
[ -f /root/.ssh/id_ed25519 ] || ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N "" -q

# Create .ssh dir on Storage Box via SFTP
PUBKEY=$(cat /root/.ssh/id_ed25519.pub)
sftp -P %d -o StrictHostKeyChecking=no %s@%s << 'SFTP' || true
mkdir .ssh
SFTP

# Install public key on Storage Box
echo "$PUBKEY" | sftp -P %d -o StrictHostKeyChecking=no %s@%s:.ssh/authorized_keys || true

# Create backup directory
sftp -P %d -o StrictHostKeyChecking=no %s@%s << 'SFTP' || true
mkdir %s
SFTP

# Test connection
ssh -p %d -o StrictHostKeyChecking=no -o BatchMode=yes %s@%s "echo OK" 2>/dev/null && echo "STORAGEBOX_SSH_OK" || echo "STORAGEBOX_SSH_FAIL"
`,
		sb.Port, sb.User, sb.Host,
		sb.Port, sb.User, sb.Host,
		sb.Port, sb.User, sb.Host, sb.Path,
		sb.Port, sb.User, sb.Host)

	out, err := client.Run(script)
	if err != nil {
		return fmt.Errorf("storage box SSH setup failed: %w", err)
	}

	if !strings.Contains(out, "STORAGEBOX_SSH_OK") {
		ui.Warn("Storage Box SSH key installed but connection test failed — you may need to add the key manually via Hetzner Robot panel")
	}
	return nil
}
