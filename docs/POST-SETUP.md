# Post-Setup Guide

After running `swarmforge up`, complete these steps:

## 1. DNS Configuration

All services are accessed through Traefik, which runs on the **infra node**. Every domain must point to the infra node's public IP.

Generate the exact DNS record list for your config:
```bash
swarmforge dns export
```

### Option A: Wildcard DNS (simplest)

```
A  *.example.com     → <infra-public-ip>
A  example.com       → <infra-public-ip>
```

This covers everything in one record. Recommended if you control the domain.

### Option B: Individual Records (15 total)

**Public domains (4)** — accessible from the internet:

| Record | Domain | Service |
|--------|--------|---------|
| A | `app1.example.com` | Application 1 |
| A | `app2.example.com` | Application 2 |
| A | `app3.example.com` | Application 3 |
| A | `api.example.com` | API endpoint |

**VPN-only domains (11)** — accessible only via NetBird VPN, but still need public DNS for TLS:

| Record | Domain | Service |
|--------|--------|---------|
| A | `portainer.example.com` | Portainer CE (container management) |
| A | `grafana.example.com` | Grafana (dashboards) |
| A | `minio.example.com` | MinIO Console (object storage UI) |
| A | `s3.example.com` | MinIO S3 API endpoint |
| A | `plane.example.com` | Plane.so (project management) |
| A | `openpanel.example.com` | OpenPanel (analytics) |
| A | `traefik.example.com` | Traefik Dashboard |
| A | `prometheus.example.com` | Prometheus (metrics) |
| A | `registry.example.com` | Docker Registry UI |
| A | `status-admin.example.com` | OpenStatus Dashboard |
| A | `status.example.com` | OpenStatus Public Page |

> **Note:** VPN-only domains still need public DNS A records because Let's Encrypt
> needs to reach Traefik for HTTP challenge validation. Access control is enforced
> by Traefik's `vpn-whitelist` middleware, not by DNS.

## 2. NetBird VPN Setup

1. Access the NetBird dashboard
2. Create a setup key for your team
3. Install NetBird client on developer laptops
4. Connect using: `netbird up --setup-key <key>`

## 3. Configure Internal DNS on Laptops

Add to NetBird DNS configuration or local DNS:
- Point `*.internal` to the infra node's private IP (10.0.0.1)

## 4. Verify Services

```bash
swarmforge status    # Full cluster report
```

Check each service:
- **Portainer**: https://portainer.example.com (set admin password on first visit)
- **Grafana**: https://grafana.example.com (default: admin/admin)
- **Prometheus**: https://prometheus.example.com
- **MinIO**: https://minio.example.com

## 5. Grafana Dashboards

Import recommended dashboards:
- **Node Exporter Full**: ID `1860`
- **Docker Swarm**: ID `12633`
- **PostgreSQL**: ID `9628`
- **Redis**: ID `11835`

## 6. Backup Verification

```bash
# Run a manual backup
swarmforge backup run

# Verify it was created
swarmforge backup list

# Test offsite sync (if configured)
swarmforge backup run --offsite
```

## 7. Alert Verification

1. Check Alertmanager: https://prometheus.example.com (Alerts tab)
2. Verify Slack integration: trigger a test alert or check the `#alerts` channel
3. Silence any noisy alerts during initial setup

## 8. Registry Setup

```bash
# Registry credentials are in .credentials file
cat .credentials | grep registry

# Test docker push
docker tag myimage:latest registry.example.com/myimage:latest
docker push registry.example.com/myimage:latest
```

## 9. Deploy Your Application

1. Add your app's docker-compose.yml to `stacks/`
2. Add a route in `traefik/dynamic/routes.yml`
3. Deploy: `swarmforge stack deploy <name>`
4. Logs are automatically collected by Promtail

## Credentials

All generated credentials are saved in `.credentials` (chmod 600):
- PostgreSQL password
- Redis password
- MinIO root password
- Registry password
- Swarm join tokens (worker + manager)
- Server public IPs
