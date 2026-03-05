#!/bin/bash
# SwarmForge Node Setup Script
# This script is executed on each node during 'swarmforge up'
# It is included as a reference — the Go CLI handles execution internally.

set -euo pipefail

echo "=== SwarmForge Node Setup ==="

# Update and install packages
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
  curl wget git jq htop iotop \
  net-tools dnsutils iputils-ping \
  ca-certificates gnupg lsb-release \
  software-properties-common \
  apache2-utils rsync \
  unattended-upgrades apt-listchanges \
  fail2ban ufw

# Install Docker
if ! command -v docker &>/dev/null; then
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker
  systemctl start docker
fi

# SSH Hardening
sed -i 's/#\?PasswordAuthentication .*/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/#\?PermitRootLogin .*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
sed -i 's/#\?MaxAuthTries .*/MaxAuthTries 3/' /etc/ssh/sshd_config
systemctl reload sshd || systemctl reload ssh

# Fail2ban
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
systemctl restart fail2ban

# Sysctl tuning
cat > /etc/sysctl.d/99-swarmforge.conf << 'SYSCTL'
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_probes = 5
net.ipv4.ip_local_port_range = 1024 65535
vm.swappiness = 10
vm.overcommit_memory = 1
fs.file-max = 2097152
fs.inotify.max_user_watches = 524288
net.ipv4.conf.all.forwarding = 1
SYSCTL
sysctl -p /etc/sysctl.d/99-swarmforge.conf 2>/dev/null || true

# Create data directories
mkdir -p /opt/data /opt/backups /opt/configs /opt/stacks /opt/scripts \
  /opt/traefik/acme /opt/traefik/dynamic

# Auto-updates
cat > /etc/apt/apt.conf.d/20auto-upgrades << 'CONF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
CONF
systemctl enable unattended-upgrades

echo "=== Node Setup Complete ==="
