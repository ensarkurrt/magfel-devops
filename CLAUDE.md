# SwarmForge — Claude Code Agent Guide

## What This Project Does

SwarmForge is a Go CLI tool that provisions and manages Docker Swarm clusters on Hetzner Cloud. It automates the entire lifecycle: server creation, Docker Swarm setup, 16-stack deployment, networking, security hardening, monitoring, backup, and DNS management.

## Architecture Decisions

### Why Traefik File Provider (not Compose Labels)
All route definitions live in `traefik/dynamic/routes.yml` and middleware definitions in `traefik/dynamic/middlewares.yml`. This centralizes routing configuration, makes it auditable, and avoids scattering route definitions across 16 compose files. The file provider watches for changes and hot-reloads.

### Why Swarm DNS vs CoreDNS (dual DNS)
- **Swarm built-in DNS**: Container-to-container communication within the `backend` overlay network. Service names resolve automatically (e.g., `data-postgresql_postgresql`).
- **CoreDNS**: Internal DNS for laptop/VPN access. Maps human-friendly names like `pg.internal` → `10.0.0.2`. Used by developers connecting via NetBird VPN.

### Why `volatile-lru` (not `allkeys-lru`)
Redis uses `volatile-lru` eviction policy because:
- OTP tokens have TTL set → they are TTL-based but CRITICAL
- `allkeys-lru` would evict ANY key under memory pressure, including non-TTL keys
- `volatile-lru` only evicts keys WITH a TTL, preserving permanent keys
- When memory is full, least-recently-used TTL keys are evicted first

### Why hcloud CLI (not HTTP API)
The `hcloud` CLI provides a clean, well-tested interface for Hetzner Cloud operations. It handles authentication, retries, and output formatting. The Go code wraps it as a subprocess and parses JSON output.

## Project Structure

```
cmd/           → Cobra CLI commands (one file per command group)
internal/
  config/      → YAML config parsing and validation
  ui/          → Colored terminal output, spinners, status display
  ssh/         → SSH client wrapper, key generation
  hetzner/     → hcloud CLI wrapper (server, network, SSH key management)
  security/    → UFW, fail2ban, SSH hardening, sysctl tuning
  swarm/       → Docker Swarm init/join, node labels, secrets, overlay networks
  deploy/      → Stack deployment, health checks, deployment order
  dns/         → CoreDNS hosts file generation, DNS record management
  backup/      → Backup orchestration, offsite sync, cron setup
stacks/        → 18 Docker Compose stack files (one directory per stack)
traefik/       → Traefik dynamic configuration (routes + middlewares)
scripts/       → Helper shell scripts
templates/     → Template files (Grafana datasources, etc.)
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `swarmforge init` | Interactive config wizard → creates swarmforge.yml |
| `swarmforge up` | Full infrastructure provisioning (19 steps) |
| `swarmforge down` | Destroy all infrastructure (with confirmation) |
| `swarmforge status` | Detailed cluster status report |
| `swarmforge node list\|add\|remove\|promote\|demote` | Swarm node management |
| `swarmforge stack list\|deploy\|remove\|update\|logs` | Stack management |
| `swarmforge dns list\|add\|remove\|regenerate` | Internal DNS records |
| `swarmforge backup run [--offsite]\|list\|restore` | Backup operations |
| `swarmforge firewall status\|apply` | UFW management |
| `swarmforge secret list\|set\|remove` | Docker secrets |
| `swarmforge ssh <node>` | SSH into a node |
| `swarmforge logs <service>` | Service log viewing |
| `swarmforge config validate\|show` | Config management |

## Global Flags

- `--config <path>` — Config file (default: swarmforge.yml)
- `--verbose` / `-v` — Detailed output
- `--dry-run` — Preview changes without executing

## How to Add a New Stack

1. Create `stacks/<stack-name>/docker-compose.yml`
2. Add the stack name to `internal/deploy/order.go` → `DeploymentOrder`
3. Add route in `traefik/dynamic/routes.yml` (with appropriate middleware)
4. If it needs a database, add creation logic in `cmd/up.go` → `createAdditionalDatabases`
5. If VPN-only, use `vpn-whitelist` middleware in the route

## How to Add a New Route

Edit `traefik/dynamic/routes.yml`:
```yaml
my-service:
  rule: "Host(`myservice.example.com`)"
  service: my-service-svc
  entryPoints: [websecure]
  tls:
    certResolver: letsencrypt
  middlewares: [vpn-whitelist, secure-headers]  # or just [secure-headers] for public
```

## How to Add an Alert Rule

Edit `stacks/mon-prometheus/alert-rules.yml` and add a new rule under the appropriate group.

## Port Map

| Service | Port | Access |
|---------|------|--------|
| Traefik (HTTP) | 80 | Public |
| Traefik (HTTPS) | 443 | Public |
| CoreDNS | 53 | Private + VPN |
| Portainer | 9000 | VPN |
| Registry | 5000 | VPN + Private |
| PostgreSQL | 5432 | Private + VPN |
| Redis | 6379 | Private + VPN |
| MinIO API | 9000 | Private + VPN |
| MinIO Console | 9001 | VPN |
| Loki | 3100 | Private + VPN |
| Prometheus | 9090 | VPN |
| Grafana | 3000 | VPN |
| Alertmanager | 9093 | VPN |
| NetBird Signal | 10000 | Public |
| NetBird TURN | 3478 | Public |

## Network Topology

1. **Hetzner Private Network** (10.0.0.0/24) — Inter-node traffic, Swarm communication
2. **Docker Overlay** (`backend`) — All stacks connected, Swarm DNS resolution
3. **NetBird VPN Mesh** (100.64.0.0/10) — External secure access for management

## Security Layers

- SSH: Password auth disabled, MaxAuthTries 3, key-only
- UFW: Role-based rules, deny-by-default
- Fail2ban: SSH brute-force protection
- Docker Secrets: All passwords stored as Swarm secrets
- Sysctl: Kernel-level TCP/memory tuning
- Unattended-upgrades: Auto security patches

## Backup Strategy

Daily cron job backs up:
- PostgreSQL: pg_dumpall + per-DB pg_dump (custom format)
- Redis: BGSAVE + RDB snapshot + AOF directory
- MinIO: .minio.sys config
- Portainer: Full data directory
- Traefik: ACME certs + dynamic config
- CoreDNS: Config files
- Swarm: Stack compose files + service inspect

Retention: 14 days local, optional offsite to Hetzner Storage Box via rsync.

## Build & Run

```bash
make build          # Build binary
make install        # Install to GOPATH/bin
./swarmforge init   # Interactive setup
./swarmforge up     # Deploy everything
```
