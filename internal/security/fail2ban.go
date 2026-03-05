package security

import (
	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func SetupFail2ban(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `apt-get install -y -qq fail2ban
cat > /etc/fail2ban/jail.local << 'JAIL'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5
backend = systemd

[sshd]
enabled = true
port = ssh
filter = sshd
maxretry = 3
bantime = 7200
JAIL
systemctl enable fail2ban
systemctl restart fail2ban`
	_, err := client.Run(script)
	return err
}

func GetFail2banStatus(client *sshpkg.Client) (string, error) {
	return client.Run("fail2ban-client status sshd 2>/dev/null || echo 'fail2ban not running'")
}
