package security

import (
	"fmt"
	"strings"

	"github.com/ensarkurrt/swarmforge/internal/config"
	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
)

// shellEscape escapes a string for safe use in single-quoted shell arguments.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func SetupNode(client *sshpkg.Client, node config.NodeConfig, cfg *config.Config) error {
	steps := []struct {
		name string
		fn   func(*sshpkg.Client, config.NodeConfig, *config.Config) error
	}{
		{"Setting hostname", setHostname},
		{"Installing base packages", installBasePackages},
		{"Configuring private network interface", configurePrivateNetwork},
		{"Installing Docker", installDocker},
		{"Hardening SSH", HardenSSH},
		{"Setting up fail2ban", SetupFail2ban},
		{"Applying sysctl tuning", ApplySysctl},
		{"Configuring UFW firewall", func(c *sshpkg.Client, n config.NodeConfig, cfg *config.Config) error {
			return ApplyUFW(c, n, cfg)
		}},
		{"Creating data directories", createDataDirs},
		{"Disabling DNS stub listener", func(c *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
			return disableDNSStubListener(c)
		}},
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

func setHostname(client *sshpkg.Client, node config.NodeConfig, _ *config.Config) error {
	_, err := client.Run(fmt.Sprintf("hostnamectl set-hostname '%s'", shellEscape(node.Name)))
	return err
}

func installBasePackages(client *sshpkg.Client, _ config.NodeConfig, _ *config.Config) error {
	script := `export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
  curl wget git jq htop iotop \
  net-tools dnsutils iputils-ping \
  ca-certificates gnupg lsb-release \
  software-properties-common \
  apache2-utils rsync ufw \
  unattended-upgrades apt-listchanges`
	_, err := client.Run(script)
	return err
}

func configurePrivateNetwork(client *sshpkg.Client, node config.NodeConfig, _ *config.Config) error {
	if node.PrivateIP == "" {
		return nil
	}
	// Hetzner Cloud assigns private IPs via DHCP with /32 mask and routes
	// through a gateway (e.g. 10.0.0.1). We must NOT add a static /24 address
	// because that creates a local route that bypasses the gateway, breaking
	// inter-node communication. Instead, just ensure the interface is UP and
	// let Hetzner's DHCP handle the IP and routing.
	script := `
# Hetzner private network interface is ens10 on most images; fallback to others
IFACE=""
for candidate in ens10 enp7s0 eth1; do
  if ip link show "$candidate" &>/dev/null 2>&1; then
    IFACE="$candidate"
    break
  fi
done

# Last resort: any interface that isn't eth0, lo, or virtual
if [ -z "$IFACE" ]; then
  IFACE=$(ip link show | awk -F': ' '/^[0-9]+: /{print $2}' \
    | sed 's/@.*//' \
    | grep -vE '^(eth0|lo|docker|veth|br-|vxlan|overlay|wg)' | head -1)
fi

if [ -z "$IFACE" ]; then
  echo "Private network interface not found, skipping"
  exit 0
fi

echo "Configuring private network on interface: $IFACE"
ip link set "$IFACE" up
ip link set "$IFACE" mtu 1450

# Remove any conflicting static netplan config that overrides DHCP routing
rm -f /etc/netplan/60-hetzner-private.yaml
netplan apply 2>/dev/null || true

# Wait for DHCP to assign IP and routes
sleep 3
`
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
	script := `mkdir -p /opt/data /opt/backups /opt/configs /opt/stacks /opt/traefik/acme /opt/traefik/dynamic /var/log/traefik`
	_, err := client.Run(script)
	return err
}

// disableDNSStubListener frees port 53 so CoreDNS can bind to it.
// systemd-resolved's stub listener occupies 127.0.0.53:53 by default,
// which conflicts with host-mode port publishing for CoreDNS.
func disableDNSStubListener(client *sshpkg.Client) error {
	script := `
sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf
grep -q '^DNSStubListener=no' /etc/systemd/resolved.conf || echo 'DNSStubListener=no' >> /etc/systemd/resolved.conf
systemctl restart systemd-resolved
ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
`
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
	script := fmt.Sprintf(`curl -fsSL https://pkgs.netbird.io/install.sh | sh
netbird up --setup-key '%s' --management-url '%s'`,
		shellEscape(setupKey), shellEscape(managementURL))
	_, err := client.Run(script)
	return err
}

// HardenSSHtoVPN updates UFW to restrict SSH access to VPN subnet only.
// Call this only after NetBird agents are confirmed connected.
func HardenSSHtoVPN(client *sshpkg.Client, vpnSubnet, privateSubnet string) error {
	script := fmt.Sprintf(`ufw delete allow 22/tcp 2>/dev/null || true
ufw allow from %s to any port 22 proto tcp comment 'SSH via VPN'
ufw allow from %s to any port 22 proto tcp comment 'SSH via private net'
ufw reload`, vpnSubnet, privateSubnet)
	_, err := client.Run(script)
	return err
}
