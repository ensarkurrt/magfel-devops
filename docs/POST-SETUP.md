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
| A | `umami.example.com` | Umami (web analytics) |
| A | `twenty.example.com` | Twenty (CRM) |
| A | `traefik.example.com` | Traefik Dashboard |
| A | `prometheus.example.com` | Prometheus (metrics) |
| A | `registry.example.com` | Docker Registry UI |
| A | `status-admin.example.com` | OpenStatus Dashboard |
| A | `status.example.com` | OpenStatus Public Page |

> **Note:** VPN-only domains still need public DNS A records because Let's Encrypt
> needs to reach Traefik for HTTP challenge validation. Access control is enforced
> by Traefik's `vpn-whitelist` middleware, not by DNS.

## 2. NetBird VPN Setup

NetBird dashboard `https://netbird.example.com` adresinde çalışır (kendi domain'inle değiştir).

### İlk Kurulum

1. DNS kaydını ekle: `netbird.example.com` → infra node public IP
2. Tarayıcıdan aç: `https://netbird.example.com`
3. İlk açılışta admin hesabı oluştur (single-account mode, OAuth gerekmez)
4. **Setup Keys** → **Create Setup Key** → key'i kopyala
5. `swarmforge.yml`'daki `netbird.setup_key` alanını güncelle

### Client Kurulumu (laptop/sunucu)

```bash
# macOS
brew install netbirdio/tap/netbird

# Linux
curl -fsSL https://pkgs.netbird.io/install.sh | sh

# Bağlan
netbird up --setup-key <setup-key> --management-url https://netbird.example.com
```

### Bağlantı Kontrolü

```bash
netbird status          # Bağlı peer'ları listeler
ping 100.64.x.x         # Diğer peer'lara ping at
```

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

## 9. CI Runner Setup (GitHub Actions)

Self-hosted runner'ları Swarm üzerine deploy edin:

```bash
# Runner stack'ini deploy et
swarmforge runner deploy \
  --repo https://github.com/your-org/your-repo \
  --pat ghp_xxxxxxxxxxxx \
  --replicas 2

# Durumu kontrol et
swarmforge runner status

# İhtiyaca göre scale et
swarmforge runner scale 4    # Yoğun dönemde
swarmforge runner scale 1    # Sakin dönemde
```

Runner'lar otomatik olarak GitHub'a kayıt olur ve `self-hosted,linux,swarm` label'ları ile görünür.

GitHub Actions workflow'larında kullanmak için:
```yaml
jobs:
  build:
    runs-on: self-hosted   # veya [self-hosted, linux, swarm]
```

> **Not:** Preview workflow'ları (`preview-deploy.yml`, `preview-cleanup.yml`) bu projede kullanılmadığından devre dışı bırakılmıştır.

## 10. Deploy Your Application

1. Add your app's docker-compose.yml to `stacks/`
2. Add a route in `traefik/dynamic/routes.yml`
3. Deploy: `swarmforge stack deploy <name>`
4. Logs are automatically collected by Promtail

## 11. Credentials

All generated credentials are saved in `.credentials` (chmod 600):
- PostgreSQL password
- Redis password
- MinIO root password
- Registry password
- Swarm join tokens (worker + manager)
- Server public IPs
