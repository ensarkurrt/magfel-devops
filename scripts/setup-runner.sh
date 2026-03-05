#!/usr/bin/env bash
# setup-runner.sh — Install one or more GitHub Actions self-hosted runners
#
# Usage:
#   ./scripts/setup-runner.sh <GITHUB_REPO_URL> <RUNNER_TOKEN> [COUNT] [NAME_PREFIX] [LABELS]
#
# Examples:
#   # Single runner
#   ./scripts/setup-runner.sh https://github.com/ensarkurrt/my-backend ABC123TOKEN
#
#   # 3 parallel runners
#   ./scripts/setup-runner.sh https://github.com/ensarkurrt/my-backend ABC123TOKEN 3
#
#   # 4 runners with custom prefix and labels
#   ./scripts/setup-runner.sh https://github.com/ensarkurrt/my-backend ABC123TOKEN 4 swarm-runner "self-hosted,linux,swarm,preview"
#
# Prerequisites:
#   - Ubuntu 22.04+ / Debian 12+
#   - Docker installed
#   - Run as root or with sudo
#
# Note: Each runner needs its own registration token. For multiple runners,
# the same token works if used before expiry (1 hour). Generate tokens via:
#   GitHub repo → Settings → Actions → Runners → New self-hosted runner

set -euo pipefail

REPO_URL="${1:?Usage: $0 <GITHUB_REPO_URL> <RUNNER_TOKEN> [COUNT] [NAME_PREFIX] [LABELS]}"
RUNNER_TOKEN="${2:?Usage: $0 <GITHUB_REPO_URL> <RUNNER_TOKEN> [COUNT] [NAME_PREFIX] [LABELS]}"
RUNNER_COUNT="${3:-1}"
RUNNER_NAME_PREFIX="${4:-$(hostname)}"
RUNNER_LABELS="${5:-self-hosted,linux,swarm}"
RUNNER_VERSION="2.322.0"
RUNNER_USER="github-runner"
RUNNER_BASE_DIR="/opt/actions-runner"

echo "=== GitHub Actions Self-Hosted Runner Setup ==="
echo "Repository  : ${REPO_URL}"
echo "Runner Count: ${RUNNER_COUNT}"
echo "Name Prefix : ${RUNNER_NAME_PREFIX}"
echo "Labels      : ${RUNNER_LABELS}"
echo ""

# Create runner user
if ! id "${RUNNER_USER}" &>/dev/null; then
    useradd -m -s /bin/bash "${RUNNER_USER}"
    echo "Created user: ${RUNNER_USER}"
fi

# Add runner user to docker group
usermod -aG docker "${RUNNER_USER}"

# Detect architecture
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)  RUNNER_ARCH="x64" ;;
    aarch64) RUNNER_ARCH="arm64" ;;
    *)       echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

RUNNER_TARBALL="actions-runner-linux-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
TARBALL_CACHE="/tmp/${RUNNER_TARBALL}"

# Download runner binary once
if [ ! -f "${TARBALL_CACHE}" ]; then
    echo "Downloading runner v${RUNNER_VERSION} (${RUNNER_ARCH})..."
    curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_TARBALL}" -o "${TARBALL_CACHE}"
fi

# Install each runner instance
for i in $(seq 1 "${RUNNER_COUNT}"); do
    if [ "${RUNNER_COUNT}" -eq 1 ]; then
        RUNNER_NAME="${RUNNER_NAME_PREFIX}"
        RUNNER_DIR="${RUNNER_BASE_DIR}"
    else
        RUNNER_NAME="${RUNNER_NAME_PREFIX}-${i}"
        RUNNER_DIR="${RUNNER_BASE_DIR}-${i}"
    fi

    echo ""
    echo "--- Setting up runner ${i}/${RUNNER_COUNT}: ${RUNNER_NAME} ---"

    # Create runner directory
    mkdir -p "${RUNNER_DIR}"

    # Extract runner
    tar xzf "${TARBALL_CACHE}" -C "${RUNNER_DIR}"
    chown -R "${RUNNER_USER}:${RUNNER_USER}" "${RUNNER_DIR}"

    # Configure runner
    echo "Configuring ${RUNNER_NAME}..."
    cd "${RUNNER_DIR}"
    sudo -u "${RUNNER_USER}" ./config.sh \
        --url "${REPO_URL}" \
        --token "${RUNNER_TOKEN}" \
        --name "${RUNNER_NAME}" \
        --labels "${RUNNER_LABELS}" \
        --work "_work" \
        --unattended \
        --replace

    # Install as systemd service
    ./svc.sh install "${RUNNER_USER}"
    ./svc.sh start

    echo "Runner ${RUNNER_NAME} started"
done

echo ""
echo "=== Setup Complete: ${RUNNER_COUNT} runner(s) installed ==="
echo ""
echo "Manage runners:"
echo "  Status:  systemctl list-units 'actions.runner.*'"
echo "  Logs:    journalctl -u actions.runner.* -f"
echo "  Stop:    cd /opt/actions-runner[-N] && ./svc.sh stop"
echo "  Remove:  cd /opt/actions-runner[-N] && ./svc.sh stop && ./config.sh remove --token <TOKEN>"
