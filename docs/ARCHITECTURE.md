# SwarmForge Architecture

## Overview

SwarmForge provisions a 4-node Docker Swarm cluster on Hetzner Cloud with a complete service stack including reverse proxy, databases, monitoring, logging, and development tools.

## Node Roles

| Node | Role | Services |
|------|------|----------|
| swarm-infra | Manager | Traefik, CoreDNS, Portainer, Registry, NetBird |
| swarm-data | Worker | PostgreSQL, Redis, MinIO |
| swarm-apps | Worker | Application workloads |
| swarm-tools | Worker | Prometheus, Grafana, Loki, Alertmanager, Plane, OpenPanel, OpenStatus |

## Network Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Internet                          │
│                   ┌──────┐                           │
│                   │ :80  │                           │
│                   │ :443 │                           │
│                   └──┬───┘                           │
├──────────────────────┼──────────────────────────────┤
│  Hetzner Cloud       │                               │
│  ┌───────────────────┼────────────────────────────┐ │
│  │ Private Network (10.0.0.0/24)                  │ │
│  │                   │                             │ │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌──────┐ │ │
│  │  │ infra  │  │  data  │  │  apps  │  │ tools│ │ │
│  │  │10.0.0.1│  │10.0.0.2│  │10.0.0.3│  │.0.0.4│ │ │
│  │  └───┬────┘  └───┬────┘  └───┬────┘  └──┬───┘ │ │
│  │      └──────┬─────┴──────┬────┴──────┬───┘     │ │
│  │             │  Docker Overlay (backend)  │      │ │
│  └─────────────┴───────────────────────────┴──────┘ │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │ NetBird VPN Mesh (100.64.0.0/10)             │   │
│  │  Developer laptops ←→ All nodes              │   │
│  └──────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
```

## Stack Dependencies

```
infra-dns ──────────────────────────────────────────┐
data-postgresql ──┐                                  │
data-redis ───────┤                                  │
data-minio ───────┘                                  │
                   │                                  │
infra-traefik ─────┤ (routes to all services)        │
infra-portainer ───┘                                  │
infra-registry ────────────────────────────────────  │
infra-netbird ─────────────────────────────────────  │
                                                      │
log-loki ──────────────────────────────────────────  │
log-promtail ──────── (global, auto-discovers all) ─┘

mon-prometheus ────── (scrapes all exporters)
mon-grafana ───────── (datasources: prometheus + loki)
mon-alertmanager ──── (receives from prometheus)

tool-plane ────────── (uses postgresql DB + redis)
tool-openpanel ────── (uses postgresql DB + redis)
tool-openstatus ───── (uses own libSQL database)
```

## Data Flow

### Request Flow (Public)
```
Client → Hetzner LB/DNS → Traefik (443) → Service container
```

### Request Flow (VPN)
```
Developer laptop → NetBird VPN → Traefik (443) → Service container
                               → CoreDNS (53) → Internal resolution
```

### Log Flow
```
Any container → Docker socket → Promtail (global) → Loki → Grafana
```

### Metrics Flow
```
Node Exporter (global) → Prometheus → Grafana
cAdvisor (global)      →            → Alertmanager → Slack
PostgreSQL Exporter    →
Redis Exporter         →
MinIO metrics          →
Traefik metrics        →
```

## Security Model

- **Public exposure**: Only ports 80 and 443 (Traefik) are open to the internet
- **VPN-only services**: Portainer, Grafana, Prometheus, MinIO Console, Registry, Plane, OpenPanel
- **Private network only**: Database ports (5432, 6379, 9000), Swarm ports (2377, 7946, 4789)
- **All passwords**: Managed as Docker secrets, auto-generated if not provided
- **SSH**: Key-only, max 3 attempts, fail2ban protection
