package security

import (
	"fmt"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
)

func SetupNode(client *sshpkg.Client, node config.NodeConfig, cfg *config.Config) error {
	steps := []struct {
		name string
		fn   func(*sshpkg.Client, config.NodeConfig, *config.Config) error
	}{
		{"Installing base packages", installBasePackages},
		{"Installing Docker", installDocker},
		{"Hardening SSH", HardenSSH},
		{"Setting up fail2ban", SetupFail2ban},
		{"Applying sysctl tuning", ApplySysctl},
		{"Configuring UFW firewall", func(c *sshpkg.Client, n config.NodeConfig, cfg *config.Config) error {
			return ApplyUFW(c, n, cfg)
		}},
		{"Creating data directories", createDataDirs},
		{"Setting up auto-updates", setupAutoUpdates},
	}

	for _, step := range steps {
		ui.Debug("  %s on %s...", step.name, node.Name)
		if err := step.fn(client, node, cfg); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}

func installBasePackages(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
  curl wget git jq htop iotop \
  net-tools dnsutils iputils-ping \
  ca-certificates gnupg lsb-release \
  software-properties-common \
  apache2-utils rsync \
  unattended-upgrades apt-listchanges`
	_, err := client.Run(script)
	return err
}

func installDocker(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `if command -v docker &>/dev/null; then
  echo "Docker already installed"
  exit 0
fi
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker`
	_, err := client.Run(script)
	return err
}

func createDataDirs(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `mkdir -p /opt/data /opt/backups /opt/configs /opt/stacks /opt/traefik/acme /opt/traefik/dynamic`
	_, err := client.Run(script)
	return err
}

func setupAutoUpdates(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `cat > /etc/apt/apt.conf.d/20auto-upgrades << 'CONF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
CONF
systemctl enable unattended-upgrades`
	_, err := client.Run(script)
	return err
}

func InstallNetbird(client *sshpkg.Client, setupKey, managementURL string) error {
	if setupKey == "" {
		ui.Debug("Skipping NetBird install (no setup key)")
		return nil
	}
	script := fmt.Sprintf(`curl -fsSL https://pkgs.netbird.io/install.sh | sh
netbird up --setup-key %s --management-url %s`, setupKey, managementURL)
	_, err := client.Run(script)
	return err
}
