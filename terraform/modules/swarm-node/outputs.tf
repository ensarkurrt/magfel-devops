output "public_ip" {
  description = "Server public IPv4 address"
  value       = hcloud_server.this.ipv4_address
}

output "server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.this.id
}

output "swarm_worker_token" {
  description = "Swarm worker join token (only set on manager)"
  value       = var.node_role == "manager" ? trimspace(data.external.swarm_tokens[0].result.worker) : ""
  sensitive   = true
}

output "swarm_manager_token" {
  description = "Swarm manager join token (only set on manager)"
  value       = var.node_role == "manager" ? trimspace(data.external.swarm_tokens[0].result.manager) : ""
  sensitive   = true
}
