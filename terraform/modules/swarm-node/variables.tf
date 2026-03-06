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
  description = "Private IP address in the Hetzner network"
  type        = string
}

variable "node_role" {
  description = "Node role: manager or worker"
  type        = string
  validation {
    condition     = contains(["manager", "worker"], var.node_role)
    error_message = "node_role must be 'manager' or 'worker'"
  }
}

variable "swarm_join_token" {
  description = "Docker Swarm join token (empty for manager)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "manager_ip" {
  description = "Manager private IP for Swarm join (empty for manager)"
  type        = string
  default     = ""
}

variable "node_labels" {
  description = "Docker Swarm node labels"
  type        = map(string)
  default     = {}
}

variable "manager_public_ip" {
  description = "Manager public IP for SSH label assignment (only needed for workers)"
  type        = string
  default     = ""
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key for provisioners"
  type        = string
}
