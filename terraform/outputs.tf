output "manager_ip" {
  description = "Manager node public IP"
  value       = module.manager.public_ip
}

output "manager_private_ip" {
  description = "Manager node private IP"
  value       = var.manager.private_ip
}

output "worker_ips" {
  description = "Worker nodes public IPs"
  value = {
    for name, worker in module.workers : name => {
      public_ip  = worker.public_ip
      private_ip = var.workers[name].private_ip
      labels     = var.workers[name].labels
    }
  }
}

output "ssh_command" {
  description = "SSH into manager node"
  value       = "ssh root@${module.manager.public_ip}"
}
