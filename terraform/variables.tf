variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "cluster_name" {
  description = "Cluster name prefix for all resources"
  type        = string
  default     = "swarm"
}

variable "network_zone" {
  description = "Hetzner network zone"
  type        = string
  default     = "eu-central"
}

variable "private_subnet" {
  description = "Private network CIDR"
  type        = string
  default     = "10.0.0.0/24"
}

variable "ip_offset" {
  description = "Starting IP offset in the subnet (e.g. 10 → first node gets 10.0.0.10)"
  type        = number
  default     = 10
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key"
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key (for provisioners)"
  type        = string
  default     = "~/.ssh/id_ed25519"
}

variable "domain" {
  description = "Base domain for services"
  type        = string
  default     = "example.com"
}

# ─── Phase 2: Lockdown ──────────────────────────────────────────────
# Boş bırakılırsa → Phase 1 (açık setup modu)
# Dolu verilirse  → Phase 2 (NetBird kur, UFW aç, Hetzner FW kısıtla)
#
# Akış:
#   1. terraform apply                                    → sunucular açık
#   2. NetBird UI'dan setup key al
#   3. terraform.tfvars'a yaz:  netbird_setup_key = "nb-xxx"
#   4. terraform apply                                    → kilitlenir
#
# Key bir kez tfvars'a yazıldığında kalıcıdır.
# Sonraki tüm apply'larda yeni eklenen node'lar da otomatik lockdown olur.
#
variable "netbird_setup_key" {
  description = "NetBird setup key — boş ise lockdown YAPILMAZ, dolu ise tüm node'lar kilitlenir"
  type        = string
  default     = ""
  sensitive   = true
}

# ─── Node Tanımları ──────────────────────────────────────────────────
#
# replicas = N → aynı tipten N adet node oluşturur
# IP'ler otomatik atanır, manuel IP vermeye gerek yok
# location her grup için ayrı belirlenebilir (multi-region)
#
# Örnekler:
#   replicas = 1  → swarm-mgr-infra-0
#   replicas = 3  → swarm-mgr-infra-0, swarm-mgr-infra-1, swarm-mgr-infra-2
#
variable "managers" {
  description = "Manager node groups — ilk grubun ilk node'u Swarm'ı başlatır"
  type = map(object({
    server_type = string
    location    = string
    replicas    = number
    labels      = optional(map(string), {})
  }))
  default = {
    infra = {
      server_type = "cx22"
      location    = "fsn1"
      replicas    = 1
      labels      = { "role" = "infra" }
    }
  }

  validation {
    condition     = sum([for g in var.managers : g.replicas]) > 0
    error_message = "En az 1 manager node olmalı (toplam replicas > 0)"
  }

  validation {
    condition     = alltrue([for g in var.managers : g.replicas >= 0])
    error_message = "replicas negatif olamaz"
  }
}

variable "workers" {
  description = "Worker node groups — entry ekle + terraform apply"
  type = map(object({
    server_type = string
    location    = string
    replicas    = number
    labels      = optional(map(string), {})
  }))
  default = {
    data = {
      server_type = "cpx31"
      location    = "fsn1"
      replicas    = 1
      labels      = { "role" = "data" }
    }
    apps = {
      server_type = "cx32"
      location    = "fsn1"
      replicas    = 1
      labels      = { "role" = "apps" }
    }
    tools = {
      server_type = "cx32"
      location    = "fsn1"
      replicas    = 1
      labels      = { "role" = "tools" }
    }
  }
}

# ─── Stack Deployment ─────────────────────────────────────────────────
variable "deploy_stacks" {
  description = "Stack'leri Terraform üzerinden deploy et"
  type        = bool
  default     = true
}

variable "stacks" {
  description = "Deploy edilecek stack listesi (sıralı)"
  type = list(object({
    name       = string
    source_dir = string
  }))
  default = [
    { name = "infra-dns", source_dir = "../stacks/infra-dns" },
    { name = "data-postgresql", source_dir = "../stacks/data-postgresql" },
    { name = "data-redis", source_dir = "../stacks/data-redis" },
    { name = "data-minio", source_dir = "../stacks/data-minio" },
    { name = "infra-traefik", source_dir = "../stacks/infra-traefik" },
    { name = "infra-portainer", source_dir = "../stacks/infra-portainer" },
    { name = "infra-registry", source_dir = "../stacks/infra-registry" },
    { name = "infra-netbird", source_dir = "../stacks/infra-netbird" },
    { name = "log-loki", source_dir = "../stacks/log-loki" },
    { name = "log-promtail", source_dir = "../stacks/log-promtail" },
    { name = "mon-prometheus", source_dir = "../stacks/mon-prometheus" },
    { name = "mon-grafana", source_dir = "../stacks/mon-grafana" },
    { name = "mon-alertmanager", source_dir = "../stacks/mon-alertmanager" },
    { name = "app-gowa", source_dir = "../stacks/app-gowa" },
    { name = "tool-umami", source_dir = "../stacks/tool-umami" },
    { name = "tool-openstatus", source_dir = "../stacks/tool-openstatus" },
    { name = "tool-twenty", source_dir = "../stacks/tool-twenty" },
  ]
}
