# Post-Setup Guide

After running `swarmforge up`, complete these steps:

## 1. DNS Configuration

Point your domain DNS records to the infra node's public IP:

```
A  *.example.com     → <infra-public-ip>
A  example.com       → <infra-public-ip>
```

Or for specific subdomains:
```
A  portainer.example.com    → <infra-public-ip>
A  grafana.example.com      → <infra-public-ip>
A  minio.example.com        → <infra-public-ip>
A  s3.example.com           → <infra-public-ip>
...
```

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
