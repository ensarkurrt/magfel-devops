# ─── Leader ───────────────────────────────────────────────────────────
output "leader_public_ip" {
  description = "Swarm leader (first manager) public IP"
  value       = module.manager_nodes[local.leader_key].public_ip
}

output "leader_private_ip" {
  description = "Swarm leader private IP"
  value       = local.leader_ip
}

# ─── All Managers ────────────────────────────────────────────────────
output "managers" {
  description = "All manager nodes"
  value = {
    for key, node in module.manager_nodes : key => {
      name       = node.name
      public_ip  = node.public_ip
      private_ip = local.node_ips[key]
      location   = local.manager_nodes[key].location
      labels     = local.manager_nodes[key].labels
    }
  }
}

# ─── All Workers ─────────────────────────────────────────────────────
output "workers" {
  description = "All worker nodes"
  value = {
    for key, node in module.worker_nodes : key => {
      name       = node.name
      public_ip  = node.public_ip
      private_ip = local.node_ips[key]
      location   = local.worker_nodes[key].location
      labels     = local.worker_nodes[key].labels
    }
  }
}

# ─── IP Map (all nodes) ─────────────────────────────────────────────
output "ip_map" {
  description = "All auto-assigned private IPs"
  value       = local.node_ips
}

# ─── Quick SSH ───────────────────────────────────────────────────────
output "ssh_leader" {
  description = "SSH into leader node"
  value       = "ssh root@${module.manager_nodes[local.leader_key].public_ip}"
}

# ─── Node Count ──────────────────────────────────────────────────────
output "total_nodes" {
  description = "Total node count"
  value = {
    managers = length(local.manager_nodes)
    workers  = length(local.worker_nodes)
    total    = length(local.all_node_keys)
  }
}
