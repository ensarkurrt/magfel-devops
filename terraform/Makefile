.PHONY: init plan apply destroy status ssh nodes stacks

# ─── Temel Komutlar ──────────────────────────────────────────────────

init:           ## Provider'ları indir
	terraform init

plan:           ## Değişiklikleri önizle (dry-run)
	terraform plan

apply:          ## Uygula — sunucu oluştur, Swarm kur, stack deploy
	terraform apply

destroy:        ## Her şeyi yık (DİKKAT!)
	terraform destroy

# ─── Cluster Bilgileri ───────────────────────────────────────────────

status:         ## Cluster durumu — node'lar, stack'ler, servisler
	@echo "═══ Node Sayısı ═══"
	@terraform output -json total_nodes | jq '.'
	@echo ""
	@echo "═══ IP Haritası ═══"
	@terraform output -json ip_map | jq '.'
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
	@echo "Kullanılabilir komutlar:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-16s %s\n", $$1, $$2}'
	@echo ""
	@echo "Yeni worker eklemek:"
	@echo "  1. terraform.tfvars → workers map'ine yeni block ekle"
	@echo "  2. make apply"
	@echo ""
	@echo "Worker replica artırmak:"
	@echo "  1. terraform.tfvars → ilgili worker'ın replicas değerini artır"
	@echo "  2. make apply"
	@echo ""
