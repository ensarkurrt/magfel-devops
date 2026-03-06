.PHONY: init plan apply destroy add-worker status ssh-manager

# ─── Temel Komutlar ──────────────────────────────────────────────────

## İlk kurulum — provider'ları indir
init:
	terraform init

## Değişiklikleri önizle (dry-run)
plan:
	terraform plan

## Değişiklikleri uygula — sunucu oluştur, Swarm kur
apply:
	terraform apply

## Her şeyi yık (DİKKAT!)
destroy:
	terraform destroy

# ─── Yardımcı Komutlar ──────────────────────────────────────────────

## Cluster durumunu göster
status:
	@echo "=== Terraform Outputs ==="
	@terraform output
	@echo ""
	@echo "=== Swarm Nodes ==="
	@ssh root@$$(terraform output -raw manager_ip) "docker node ls"
	@echo ""
	@echo "=== Running Stacks ==="
	@ssh root@$$(terraform output -raw manager_ip) "docker stack ls"

## Manager'a SSH
ssh-manager:
	ssh root@$$(terraform output -raw manager_ip)

## Stack deploy — kullanım: make deploy-stack STACK=infra-traefik
deploy-stack:
	@test -n "$(STACK)" || (echo "Usage: make deploy-stack STACK=<name>" && exit 1)
	scp -r ../stacks/$(STACK) root@$$(terraform output -raw manager_ip):/opt/stacks/$(STACK)
	ssh root@$$(terraform output -raw manager_ip) "docker stack deploy -c /opt/stacks/$(STACK)/docker-compose.yml $(STACK)"

## Tüm stack'leri deploy et (sıralı)
deploy-all:
	@for stack in infra-dns data-postgresql data-redis data-minio infra-traefik infra-portainer infra-registry infra-netbird log-loki log-promtail mon-prometheus mon-grafana mon-alertmanager app-gowa tool-umami tool-openstatus tool-twenty; do \
		echo "Deploying $$stack..."; \
		scp -r ../stacks/$$stack root@$$(terraform output -raw manager_ip):/opt/stacks/$$stack 2>/dev/null; \
		ssh root@$$(terraform output -raw manager_ip) "docker stack deploy -c /opt/stacks/$$stack/docker-compose.yml $$stack" 2>/dev/null; \
		echo "  ✓ $$stack deployed"; \
	done

# ─── Yeni Worker Ekleme Rehberi ──────────────────────────────────────
help-add-worker:
	@echo ""
	@echo "Yeni worker eklemek için:"
	@echo ""
	@echo "  1. terraform.tfvars dosyasındaki 'workers' map'ine yeni block ekle:"
	@echo ""
	@echo '     newnode = {'
	@echo '       server_type = "cx32"'
	@echo '       private_ip  = "10.0.0.5"'
	@echo '       labels      = { "role" = "custom" }'
	@echo '     }'
	@echo ""
	@echo "  2. terraform apply"
	@echo ""
	@echo "Bu kadar. Sunucu oluşturulur, Docker kurulur, Swarm'a katılır, label'lar atanır."
	@echo ""
