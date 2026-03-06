output "public_ip" {
  description = "Server public IPv4 address"
  value       = hcloud_server.this.ipv4_address
}

output "server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.this.id
}

output "name" {
  description = "Server name"
  value       = hcloud_server.this.name
}
