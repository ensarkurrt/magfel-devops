.PHONY: init plan apply lockdown destroy status ssh nodes stacks dns health backup help deploy-stack redeploy-all

# ─── Temel Komutlar ──────────────────────────────────────────────────

init:           ## Provider'ları indir
	terraform init

plan:           ## Değişiklikleri önizle (dry-run)
	terraform plan

apply:          ## Phase 1: Sunucu oluştur, Swarm kur, stack deploy (her şey açık)
	terraform apply

lockdown:       ## Phase 2: NetBird + UFW + Hetzner FW kısıtla
	@test -n "$(KEY)" && terraform apply -var='netbird_setup_key=$(KEY)' || echo "Kullanim: make lockdown KEY=nb-xxxxxxxxxxxx"

destroy:        ## Her şeyi yık (DİKKAT!)
	terraform destroy

# ─── Cluster Bilgileri ───────────────────────────────────────────────

status:         ## Cluster durumu — node'lar, stack'ler, lockdown
	@echo "=== Lockdown Status ==="
	@terraform output -raw lockdown_status 2>/dev/null || echo "Not yet applied"
	@echo ""
	@echo ""
	@echo "=== Node Count ==="
	@terraform output -json total_nodes 2>/dev/null | jq '.' || echo "Not yet applied"
	@echo ""
	@echo "=== Swarm Nodes ==="
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip 2>/dev/null) "docker node ls" 2>/dev/null || echo "Cannot connect"
	@echo ""
	@echo "=== Running Stacks ==="
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip 2>/dev/null) "docker stack ls" 2>/dev/null || echo "Cannot connect"

nodes:          ## Tüm node IP'lerini göster
	@echo "=== Managers ==="
	@terraform output -json managers 2>/dev/null | jq -r 'to_entries[] | "\(.key)\t\(.value.public_ip)\t\(.value.private_ip)\t\(.value.location)"' || echo "Not yet applied"
	@echo ""
	@echo "=== Workers ==="
	@terraform output -json workers 2>/dev/null | jq -r 'to_entries[] | "\(.key)\t\(.value.public_ip)\t\(.value.private_ip)\t\(.value.location)"' || echo "Not yet applied"

stacks:         ## Çalışan stack'leri listele
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) "docker stack ls"

dns:            ## CoreDNS hosts.internal içeriğini göster
	@terraform output -raw coredns_hosts

health:         ## Servis sağlık kontrolü
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) 'docker node ls && echo "" && docker stack ls && echo "" && docker service ls'

backup:         ## Manuel backup tetikle
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) "/opt/scripts/backup.sh"

# ─── SSH ─────────────────────────────────────────────────────────────

ssh:            ## Leader'a SSH bağlan
	ssh root@$$(terraform output -raw leader_public_ip)

# ─── Stack İşlemleri ─────────────────────────────────────────────────

deploy-stack:   ## Tek stack deploy — make deploy-stack STACK=infra-traefik
	@test -n "$(STACK)" && \
		scp -o StrictHostKeyChecking=no -r ../stacks/$(STACK) root@$$(terraform output -raw leader_public_ip):/opt/stacks/$(STACK) && \
		ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) \
			"docker stack deploy -c /opt/stacks/$(STACK)/docker-compose.yml $(STACK) --with-registry-auth" \
	|| echo "Kullanim: make deploy-stack STACK=<name>"

redeploy-all:   ## Tüm stack'leri yeniden deploy et
	@LEADER=$$(terraform output -raw leader_public_ip); \
	for stack in infra-dns data-postgresql data-redis data-minio infra-traefik infra-portainer infra-registry infra-netbird log-loki log-promtail mon-prometheus mon-grafana mon-alertmanager app-gowa tool-umami tool-openstatus tool-twenty; do \
		echo "Deploying $$stack..."; \
		scp -o StrictHostKeyChecking=no -r ../stacks/$$stack root@$$LEADER:/opt/stacks/$$stack 2>/dev/null; \
		ssh -o StrictHostKeyChecking=no root@$$LEADER \
			"docker stack deploy -c /opt/stacks/$$stack/docker-compose.yml $$stack --with-registry-auth" 2>/dev/null; \
		echo "  done: $$stack"; \
	done

# ─── Yardım ─────────────────────────────────────────────────────────

help:           ## Bu yardım mesajını göster
	@echo ""
	@echo "Kullanim akisi:"
	@echo "  1. make init                       -> Provider'lari indir"
	@echo "  2. make apply                      -> Phase 1: Cluster kur (acik)"
	@echo "  3. NetBird UI'dan setup key al"
	@echo "  4. make lockdown KEY=nb-xxx        -> Phase 2: Kilitle"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-16s %s\n", $$1, $$2}'
	@echo ""
