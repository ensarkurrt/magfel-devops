# SwarmForge — TODO

## Auto-Scale Runners (Cron Job)

GitHub Actions self-hosted runner'lar şu an manuel scale ediliyor (`swarmforge runner scale <N>`).
İleride queue'daki bekleyen job sayısına göre otomatik scale eden bir mekanizma eklenebilir.

### Nasıl çalışır

Her 2 dakikada bir cron job:
1. GitHub API'den bekleyen (queued) workflow run sayısını sorgular
2. Mevcut runner replica sayısıyla karşılaştırır
3. Gerekirse scale up/down yapar

### Tasarım

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Cron Job   │────→│  GitHub API  │────→│  Scale Decision  │
│  (2 min)    │     │  /actions/   │     │                  │
│             │     │  runs?status │     │  queued > idle   │
│             │     │  =queued     │     │  → scale up      │
│             │     │              │     │                  │
│             │     │              │     │  idle > queued   │
│             │     │              │     │  → scale down    │
└─────────────┘     └──────────────┘     └─────────────────┘
```

### Parametreler

| Parametre | Varsayılan | Açıklama |
|-----------|-----------|----------|
| `min_runners` | 1 | Minimum replica sayısı (asla altına düşmez) |
| `max_runners` | 8 | Maksimum replica sayısı (kaynak limiti) |
| `scale_up_threshold` | 1 | Bu kadar queued job varsa scale up |
| `scale_down_delay` | 5m | İdle runner'ları bu süre sonra kaldır |
| `check_interval` | 2m | Kontrol sıklığı |

### Implementasyon planı

1. **`internal/runner/autoscale.go`** — Auto-scale logic
   - GitHub API'den queued run sayısını çek
   - Mevcut runner replica sayısını al (`docker service inspect`)
   - Scale kararı ver: `desired = min(max_runners, max(min_runners, queued_jobs))`
   - `docker service scale` çalıştır

2. **`cmd/runner.go`** — Yeni subcommand'lar
   - `swarmforge runner autoscale enable` — Cron job'ı node'a kur
   - `swarmforge runner autoscale disable` — Cron job'ı kaldır
   - `swarmforge runner autoscale status` — Mevcut ayarları göster

3. **`swarmforge.yml`** — Config'e ekle
   ```yaml
   runner:
     autoscale:
       enabled: true
       min_replicas: 1
       max_replicas: 8
       check_interval: "2m"
       scale_down_delay: "5m"
   ```

4. **Cron job script** (`scripts/autoscale-runners.sh`)
   ```bash
   #!/bin/bash
   QUEUED=$(curl -s -H "Authorization: token $PAT" \
     "https://api.github.com/repos/$OWNER/$REPO/actions/runs?status=queued" \
     | jq '.total_count')

   CURRENT=$(docker service inspect ci-runner_runner \
     --format '{{.Spec.Mode.Replicated.Replicas}}')

   DESIRED=$(( QUEUED > MAX ? MAX : QUEUED < MIN ? MIN : QUEUED ))

   if [ "$DESIRED" != "$CURRENT" ]; then
     docker service scale ci-runner_runner=$DESIRED
   fi
   ```

### Dikkat edilecekler

- **Scale down sırasında çalışan job'ları kesmemek**: `--detach=false` ile scale down yapılmalı, Swarm draining task'lerin bitmesini bekler
- **Rate limit**: GitHub API rate limit'e dikkat (authenticated: 5000 req/hour)
- **Cooldown**: Arka arkaya scale up/down yapılmaması için cooldown süresi eklenmeli
- **Monitoring**: Prometheus metric'i olarak runner count ve queue depth export edilebilir
