.PHONY: init plan apply lockdown destroy status ssh nodes stacks help

# ─── Temel Komutlar ──────────────────────────────────────────────────

init:           ## Provider'ları indir
	terraform init

plan:           ## Değişiklikleri önizle (dry-run)
	terraform plan

apply:          ## Phase 1: Sunucu oluştur, Swarm kur, stack deploy (her şey açık)
	terraform apply

lockdown:       ## Phase 2: NetBird + UFW + Hetzner FW kısıtla
	@test -n "$(KEY)" || (echo "Kullanım: make lockdown KEY=nb-xxxxxxxxxxxx" && exit 1)
	terraform apply -var='netbird_setup_key=$(KEY)'

destroy:        ## Her şeyi yık (DİKKAT!)
	terraform destroy

# ─── Cluster Bilgileri ───────────────────────────────────────────────

status:         ## Cluster durumu — node'lar, stack'ler, lockdown
	@echo "═══ Lockdown Status ═══"
	@terraform output -raw lockdown_status
	@echo ""
	@echo ""
	@echo "═══ Node Sayısı ═══"
	@terraform output -json total_nodes | jq '.'
	@echo ""
	@echo "═══ Swarm Nodes ═══"
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) "docker node ls"
	@echo ""
	@echo "═══ Running Stacks ═══"
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) "docker stack ls"

nodes:          ## Tüm node IP'lerini göster
	@echo "═══ Managers ═══"
	@terraform output -json managers | jq -r 'to_entries[] | "\(.key)\t\(.value.public_ip)\t\(.value.private_ip)\t\(.value.location)"'
	@echo ""
	@echo "═══ Workers ═══"
	@terraform output -json workers | jq -r 'to_entries[] | "\(.key)\t\(.value.public_ip)\t\(.value.private_ip)\t\(.value.location)"'

stacks:         ## Çalışan stack'leri listele
	@ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) "docker stack ls"

dns:            ## CoreDNS hosts.internal içeriğini göster
	@terraform output -raw coredns_hosts

# ─── SSH ─────────────────────────────────────────────────────────────

ssh:            ## Leader'a SSH bağlan
	ssh root@$$(terraform output -raw leader_public_ip)

# ─── Stack İşlemleri ─────────────────────────────────────────────────

deploy-stack:   ## Tek stack deploy — make deploy-stack STACK=infra-traefik
	@test -n "$(STACK)" || (echo "Kullanım: make deploy-stack STACK=<name>" && exit 1)
	scp -o StrictHostKeyChecking=no -r ../stacks/$(STACK) root@$$(terraform output -raw leader_public_ip):/opt/stacks/$(STACK)
	ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) \
		"docker stack deploy -c /opt/stacks/$(STACK)/docker-compose.yml $(STACK) --with-registry-auth"

redeploy-all:   ## Tüm stack'leri yeniden deploy et
	@for stack in infra-dns data-postgresql data-redis data-minio infra-traefik infra-portainer infra-registry infra-netbird log-loki log-promtail mon-prometheus mon-grafana mon-alertmanager app-gowa tool-umami tool-openstatus tool-twenty; do \
		echo "Deploying $$stack..."; \
		scp -o StrictHostKeyChecking=no -r ../stacks/$$stack root@$$(terraform output -raw leader_public_ip):/opt/stacks/$$stack 2>/dev/null; \
		ssh -o StrictHostKeyChecking=no root@$$(terraform output -raw leader_public_ip) \
			"docker stack deploy -c /opt/stacks/$$stack/docker-compose.yml $$stack --with-registry-auth" 2>/dev/null; \
		echo "  done: $$stack"; \
	done

# ─── Yardım ─────────────────────────────────────────────────────────

help:           ## Bu yardım mesajını göster
	@echo ""
	@echo "Kullanım akışı:"
	@echo "  1. make init                       → Provider'ları indir"
	@echo "  2. make apply                      → Phase 1: Cluster kur (açık)"
	@echo "  3. NetBird UI'dan setup key al"
	@echo "  4. make lockdown KEY=nb-xxx        → Phase 2: Kilitle"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-16s %s\n", $$1, $$2}'
	@echo ""
