#!/usr/bin/env bash
# scale-runners.sh — Scale GitHub Actions runners on Docker Swarm
#
# Usage:
#   ./scripts/scale-runners.sh <COMMAND> [ARGS]
#
# Commands:
#   build                       Build runner Docker image and push to registry
#   deploy <repo_url> <pat>     Deploy runner stack with N replicas
#   scale <count>               Scale runners up or down
#   status                      Show current runner count and status
#   logs                        Follow runner logs
#   remove                      Remove runner stack entirely
#
# Examples:
#   ./scripts/scale-runners.sh build
#   ./scripts/scale-runners.sh deploy https://github.com/ensarkurrt/my-backend ghp_xxxx
#   ./scripts/scale-runners.sh scale 5      # scale up to 5 runners
#   ./scripts/scale-runners.sh scale 1      # scale down to 1
#   ./scripts/scale-runners.sh status
#   ./scripts/scale-runners.sh remove

set -euo pipefail

STACK_NAME="ci-runner"
STACK_DIR="$(cd "$(dirname "$0")/../stacks/ci-runner" && pwd)"
REGISTRY="${REGISTRY:-registry:5000}"
IMAGE="${REGISTRY}/github-runner:latest"

cmd_build() {
    echo "Building runner image..."
    docker build -t "${IMAGE}" "${STACK_DIR}"
    docker push "${IMAGE}"
    echo "Image pushed: ${IMAGE}"
}

cmd_deploy() {
    local repo_url="${1:?Usage: scale-runners.sh deploy <GITHUB_REPO_URL> <GITHUB_PAT> [REPLICAS]}"
    local pat="${2:?Usage: scale-runners.sh deploy <GITHUB_REPO_URL> <GITHUB_PAT> [REPLICAS]}"
    local replicas="${3:-2}"

    echo "Deploying ${replicas} runner(s)..."

    GITHUB_REPO_URL="${repo_url}" \
    GITHUB_PAT="${pat}" \
    REGISTRY="${REGISTRY}" \
    RUNNER_REPLICAS="${replicas}" \
    docker stack deploy -c "${STACK_DIR}/docker-compose.yml" "${STACK_NAME}"

    echo "Stack deployed. Runners will auto-register with GitHub."
    echo ""
    echo "Scale anytime with:"
    echo "  ./scripts/scale-runners.sh scale <count>"
}

cmd_scale() {
    local count="${1:?Usage: scale-runners.sh scale <count>}"

    echo "Scaling to ${count} runner(s)..."
    docker service scale "${STACK_NAME}_runner=${count}"
    echo ""
    cmd_status
}

cmd_status() {
    echo "=== Runner Status ==="
    echo ""

    # Service info
    docker service ls --filter "name=${STACK_NAME}" 2>/dev/null || echo "Stack not deployed"
    echo ""

    # Task details
    docker service ps "${STACK_NAME}_runner" --format "table {{.ID}}\t{{.Name}}\t{{.CurrentState}}\t{{.Node}}" 2>/dev/null || true
}

cmd_logs() {
    docker service logs "${STACK_NAME}_runner" --follow --tail 50
}

cmd_remove() {
    echo "Removing runner stack..."
    docker stack rm "${STACK_NAME}"
    echo "Stack removed. Runners will deregister from GitHub."
}

# Main
COMMAND="${1:-help}"
shift || true

case "${COMMAND}" in
    build)   cmd_build "$@" ;;
    deploy)  cmd_deploy "$@" ;;
    scale)   cmd_scale "$@" ;;
    status)  cmd_status "$@" ;;
    logs)    cmd_logs "$@" ;;
    remove)  cmd_remove "$@" ;;
    *)
        echo "Usage: $0 <command> [args]"
        echo ""
        echo "Commands:"
        echo "  build                       Build runner Docker image"
        echo "  deploy <repo_url> <pat>     Deploy runner stack"
        echo "  scale <count>               Scale runners up or down"
        echo "  status                      Show runner status"
        echo "  logs                        Follow runner logs"
        echo "  remove                      Remove runner stack"
        exit 1
        ;;
esac
