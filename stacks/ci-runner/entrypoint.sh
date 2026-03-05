#!/usr/bin/env bash
# entrypoint.sh — Ephemeral GitHub Actions runner
# Registers, runs ONE job, then exits. Swarm restarts the container.
set -euo pipefail

REPO_URL="${GITHUB_REPO_URL:?GITHUB_REPO_URL is required}"
RUNNER_LABELS="${RUNNER_LABELS:-self-hosted,linux,swarm}"
RUNNER_GROUP="${RUNNER_GROUP:-default}"

# Get a registration token from GitHub API using a PAT
get_registration_token() {
    # Extract owner/repo from URL
    local repo_path
    repo_path=$(echo "${REPO_URL}" | sed 's|https://github.com/||')

    curl -s -X POST \
        -H "Authorization: token ${GITHUB_PAT}" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/${repo_path}/actions/runners/registration-token" \
        | jq -r '.token'
}

# If a PAT is provided, auto-generate registration token
if [ -n "${GITHUB_PAT:-}" ]; then
    echo "Generating registration token from PAT..."
    RUNNER_TOKEN=$(get_registration_token)
elif [ -n "${RUNNER_TOKEN:-}" ]; then
    echo "Using provided runner token..."
else
    echo "ERROR: Either GITHUB_PAT or RUNNER_TOKEN must be set"
    exit 1
fi

RUNNER_NAME="swarm-runner-$(hostname)-$(date +%s)"

echo "Configuring ephemeral runner: ${RUNNER_NAME}"
./config.sh \
    --url "${REPO_URL}" \
    --token "${RUNNER_TOKEN}" \
    --name "${RUNNER_NAME}" \
    --labels "${RUNNER_LABELS}" \
    --work "_work" \
    --unattended \
    --ephemeral \
    --replace

# Cleanup on exit
cleanup() {
    echo "Removing runner registration..."
    ./config.sh remove --token "${RUNNER_TOKEN}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "Starting runner (ephemeral — will exit after one job)..."
./run.sh
