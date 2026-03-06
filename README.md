# SwarmForge

Turnkey Docker Swarm cluster management on Hetzner Cloud. SwarmForge provisions servers, sets up Docker Swarm, deploys 18 production-ready stacks, and configures networking, security, monitoring, backups, and CI/CD runners — all from a single command.

## Quick Start

```bash
# Build
make build

# Interactive setup
./swarmforge init

# Deploy everything
./swarmforge up

# Check status
./swarmforge status
```

## What It Deploys

**Infrastructure**: Traefik v3.1 (reverse proxy), CoreDNS (internal DNS), Portainer CE, Docker Registry v2, NetBird VPN

**Data Services**: PostgreSQL 16, Redis 7 (volatile-lru), MinIO S3

**Monitoring**: Prometheus + Node Exporter + cAdvisor, Grafana, Loki + Promtail (auto log collection), Alertmanager (Slack alerts)

**Tools**: Umami (web analytics), OpenStatus (uptime monitoring), Twenty (CRM)

**CI/CD**: GitHub Actions self-hosted runners (scalable Swarm service)

## Architecture

| Node | Type | Specs | Fiyat | Rol |
|------|------|-------|-------|-----|
| swarm-infra | CX23 | 2 vCPU, 4 GB, 40 GB | ~€3.49/ay | Manager — routing, DNS, VPN |
| swarm-data | CPX32 | 4 vCPU, 8 GB, 160 GB | ~€10.99/ay | Worker — databases |
| swarm-apps | CX33 | 4 vCPU, 8 GB, 80 GB | ~€5.49/ay | Worker — apps, CI runners |
| swarm-tools | CX33 | 4 vCPU, 8 GB, 80 GB | ~€5.49/ay | Worker — monitoring, tools |

**Toplam:** ~€25.46/ay (14 vCPU, 28 GB RAM, 360 GB disk)

- 3-layer networking: Hetzner private network, Docker overlay, NetBird VPN mesh
- Security: UFW firewall, fail2ban, SSH hardening, Docker secrets, kernel tuning
- Traefik file provider routing (no compose labels)
- Automatic log collection via Promtail docker_sd_configs

## Commands

```
swarmforge init                    Interactive setup
swarmforge up                      Provision everything
swarmforge down                    Tear down (with confirmation)
swarmforge status                  Detailed cluster report

swarmforge node list|add|remove|promote|demote
swarmforge stack list|deploy|remove|update|logs
swarmforge dns list|add|remove|regenerate
swarmforge backup run [--offsite]|list|restore
swarmforge firewall status|apply
swarmforge secret list|set|remove
swarmforge runner deploy|scale|status|logs|remove
swarmforge ssh <node-name>
swarmforge logs <service-name>
swarmforge config validate|show
swarmforge version
```

## Configuration

All settings in `swarmforge.yml`. See `swarmforge.yml.example` for the full reference.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — Network topology, stack dependencies, data flow
- [Post-Setup Guide](docs/POST-SETUP.md) — DNS config, VPN setup, dashboard access
- [CLAUDE.md](CLAUDE.md) — Agent guide with commands, port map, how-to guides

## Requirements

- Go 1.21+
- `hcloud` CLI installed ([install](https://github.com/hetznercloud/cli))
- Hetzner Cloud API token
- SSH key pair

## License

MIT
