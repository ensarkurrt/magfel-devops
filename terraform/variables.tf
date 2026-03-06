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

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "fsn1"
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

# ─── Node Definitions ────────────────────────────────────────────────
# Manager node — tek node, Swarm manager
variable "manager" {
  description = "Manager node configuration"
  type = object({
    server_type = string
    private_ip  = string
  })
  default = {
    server_type = "cx22"
    private_ip  = "10.0.0.1"
  }
}

# Worker nodes — istediğin kadar ekle, `terraform apply` çalıştır
variable "workers" {
  description = "Worker nodes — add a new entry and run terraform apply"
  type = map(object({
    server_type = string
    private_ip  = string
    labels      = map(string)
  }))
  default = {
    data = {
      server_type = "cpx31"
      private_ip  = "10.0.0.2"
      labels      = { "role" = "data" }
    }
    apps = {
      server_type = "cx32"
      private_ip  = "10.0.0.3"
      labels      = { "role" = "apps" }
    }
    tools = {
      server_type = "cx32"
      private_ip  = "10.0.0.4"
      labels      = { "role" = "tools" }
    }
    # ──────────────────────────────────────────────────────
    # Yeni worker eklemek için buraya bir block ekle:
    #
    # gpu = {
    #   server_type = "cx42"
    #   private_ip  = "10.0.0.5"
    #   labels      = { "role" = "gpu" }
    # }
    #
    # Sonra: terraform apply
    # ──────────────────────────────────────────────────────
  }
}

variable "domain" {
  description = "Base domain for services"
  type        = string
  default     = "example.com"
}
