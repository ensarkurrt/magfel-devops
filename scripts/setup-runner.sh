#!/usr/bin/env bash
# setup-runner.sh — Install GitHub Actions self-hosted runner on a Swarm node
#
# Usage:
#   ./scripts/setup-runner.sh <GITHUB_REPO_URL> <RUNNER_TOKEN>
#
# Example:
#   ./scripts/setup-runner.sh https://github.com/ensarkurrt/my-backend ABC123TOKEN
#
# Prerequisites:
#   - Ubuntu 22.04+ / Debian 12+
#   - Docker installed
#   - Run as root or with sudo

set -euo pipefail

REPO_URL="${1:?Usage: $0 <GITHUB_REPO_URL> <RUNNER_TOKEN>}"
RUNNER_TOKEN="${2:?Usage: $0 <GITHUB_REPO_URL> <RUNNER_TOKEN>}"
RUNNER_NAME="${3:-$(hostname)}"
RUNNER_LABELS="${4:-self-hosted,linux,swarm}"
RUNNER_VERSION="2.322.0"
RUNNER_USER="github-runner"
RUNNER_DIR="/opt/actions-runner"

echo "=== GitHub Actions Self-Hosted Runner Setup ==="
echo "Repository  : ${REPO_URL}"
echo "Runner Name : ${RUNNER_NAME}"
echo "Labels      : ${RUNNER_LABELS}"
echo ""

# Create runner user
if ! id "${RUNNER_USER}" &>/dev/null; then
    useradd -m -s /bin/bash "${RUNNER_USER}"
    echo "Created user: ${RUNNER_USER}"
fi

# Add runner user to docker group
usermod -aG docker "${RUNNER_USER}"

# Create runner directory
mkdir -p "${RUNNER_DIR}"
chown "${RUNNER_USER}:${RUNNER_USER}" "${RUNNER_DIR}"

# Download runner
cd "${RUNNER_DIR}"
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)  RUNNER_ARCH="x64" ;;
    aarch64) RUNNER_ARCH="arm64" ;;
    *)       echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

RUNNER_TARBALL="actions-runner-linux-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
if [ ! -f "${RUNNER_TARBALL}" ]; then
    echo "Downloading runner v${RUNNER_VERSION} (${RUNNER_ARCH})..."
    curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_TARBALL}" -o "${RUNNER_TARBALL}"
    tar xzf "${RUNNER_TARBALL}"
    chown -R "${RUNNER_USER}:${RUNNER_USER}" "${RUNNER_DIR}"
fi

# Configure runner
echo "Configuring runner..."
sudo -u "${RUNNER_USER}" ./config.sh \
    --url "${REPO_URL}" \
    --token "${RUNNER_TOKEN}" \
    --name "${RUNNER_NAME}" \
    --labels "${RUNNER_LABELS}" \
    --work "_work" \
    --unattended \
    --replace

# Install as systemd service
echo "Installing systemd service..."
./svc.sh install "${RUNNER_USER}"
./svc.sh start

echo ""
echo "=== Runner Setup Complete ==="
echo "Service: actions.runner.*.service"
echo "Status:  sudo ./svc.sh status"
echo "Logs:    journalctl -u actions.runner.* -f"
