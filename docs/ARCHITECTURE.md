# SwarmForge Architecture

## Overview

SwarmForge provisions a 4-node Docker Swarm cluster on Hetzner Cloud with a complete service stack including reverse proxy, databases, monitoring, logging, development tools, and CI/CD runners.

## Node Roles

| Node | Role | Type | Specs | Price | Services |
|------|------|------|-------|-------|----------|
| swarm-infra | Manager | CX22 | 2 vCPU, 4 GB RAM, 40 GB disk | ~€3.49/ay | Traefik, CoreDNS, Portainer, Registry, NetBird |
| swarm-data | Worker | CPX31 | 4 vCPU, 8 GB RAM, 160 GB disk | ~€10.99/ay | PostgreSQL, Redis, MinIO |
| swarm-apps | Worker | CX32 | 4 vCPU, 8 GB RAM, 80 GB disk | ~€5.49/ay | Application workloads (app-gowa), CI runners |
| swarm-tools | Worker | CX32 | 4 vCPU, 8 GB RAM, 80 GB disk | ~€5.49/ay | Prometheus, Grafana, Loki, Alertmanager, Umami, OpenStatus, Twenty |

**Toplam:** 14 vCPU, 28 GB RAM, 360 GB disk — ~€25.46/ay

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

app-gowa ─────────── (application workload)

tool-umami ───────── (web analytics)
tool-openstatus ───── (uptime monitoring, own libSQL database)
tool-twenty ──────── (CRM tool)

ci-runner ────────── (GitHub Actions self-hosted runners)
                      (scalable replicas, registers with GitHub)
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

### CI/CD Flow
```
GitHub push/PR → GitHub Actions → Queue job
                                    ↓
                 ci-runner replicas (Swarm service) ← swarmforge runner scale
                                    ↓
                 Build → Push to Registry → Deploy via swarmforge stack
```

## Security Model

- **Public exposure**: Only ports 80 and 443 (Traefik) are open to the internet
- **VPN-only services**: Portainer, Grafana, Prometheus, MinIO Console, Registry, Umami, Twenty
- **CI runners**: Self-hosted GitHub Actions runners, `ci-runner` stack, private network only
- **Preview workflows**: Devre dışı — bu projede preview ortamları kullanılmıyor
- **Private network only**: Database ports (5432, 6379, 9000), Swarm ports (2377, 7946, 4789)
- **All passwords**: Managed as Docker secrets, auto-generated if not provided
- **SSH**: Key-only, max 3 attempts, fail2ban protection
