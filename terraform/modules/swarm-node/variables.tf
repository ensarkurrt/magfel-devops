variable "name" {
  description = "Server name"
  type        = string
}

variable "server_type" {
  description = "Hetzner server type (cx22, cpx31, cx32, etc.)"
  type        = string
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
}

variable "ssh_key_id" {
  description = "Hetzner SSH key resource ID"
  type        = string
}

variable "network_id" {
  description = "Hetzner private network ID"
  type        = number
}

variable "subnet_id" {
  description = "Hetzner subnet ID (for dependency ordering)"
  type        = string
}

variable "private_ip" {
  description = "Private IP address (auto-assigned by main.tf)"
  type        = string
}

variable "firewall_id" {
  description = "Hetzner firewall ID to attach"
  type        = number
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key for provisioners"
  type        = string
}
