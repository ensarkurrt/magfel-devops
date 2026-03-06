# ─── Cloud-init: OS kurulumu, Docker, güvenlik ──────────────────────
data "cloudinit_config" "this" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    content      = file("${path.module}/cloud-init.yml")
  }
}

# ─── Server ──────────────────────────────────────────────────────────
resource "hcloud_server" "this" {
  name        = var.name
  server_type = var.server_type
  image       = "ubuntu-22.04"
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  user_data   = data.cloudinit_config.this.rendered

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  lifecycle {
    ignore_changes = [user_data] # cloud-init sadece ilk boot'ta çalışır
  }
}

# ─── Private Network Attachment ──────────────────────────────────────
resource "hcloud_server_network" "this" {
  server_id  = hcloud_server.this.id
  network_id = var.network_id
  ip         = var.private_ip

  depends_on = [var.subnet_id]
}

# ─── Wait for cloud-init to finish ──────────────────────────────────
resource "null_resource" "wait_for_cloudinit" {
  depends_on = [hcloud_server_network.this]

  connection {
    type        = "ssh"
    host        = hcloud_server.this.ipv4_address
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
    timeout     = "5m"
  }

  provisioner "remote-exec" {
    inline = [
      "echo 'Waiting for cloud-init to complete...'",
      "cloud-init status --wait",
      "echo 'Cloud-init finished. Docker version:'",
      "docker --version",
    ]
  }
}

# ─── Swarm Init (only on manager) ───────────────────────────────────
resource "null_resource" "swarm_init" {
  count      = var.node_role == "manager" ? 1 : 0
  depends_on = [null_resource.wait_for_cloudinit]

  connection {
    type        = "ssh"
    host        = hcloud_server.this.ipv4_address
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      "docker swarm init --advertise-addr ${var.private_ip} 2>/dev/null || echo 'Swarm already initialized'",
      "docker network create --driver overlay --attachable backend 2>/dev/null || echo 'Network already exists'",
    ]
  }
}

# ─── Get Swarm tokens from manager ──────────────────────────────────
data "external" "swarm_tokens" {
  count   = var.node_role == "manager" ? 1 : 0
  program = ["bash", "-c", <<-EOF
    WORKER=$(ssh -o StrictHostKeyChecking=no -i ${pathexpand(var.ssh_private_key_path)} root@${hcloud_server.this.ipv4_address} "docker swarm join-token -q worker")
    MANAGER=$(ssh -o StrictHostKeyChecking=no -i ${pathexpand(var.ssh_private_key_path)} root@${hcloud_server.this.ipv4_address} "docker swarm join-token -q manager")
    echo "{\"worker\": \"$WORKER\", \"manager\": \"$MANAGER\"}"
  EOF
  ]

  depends_on = [null_resource.swarm_init]
}

# ─── Swarm Join (only on workers) ───────────────────────────────────
resource "null_resource" "swarm_join" {
  count      = var.node_role == "worker" ? 1 : 0
  depends_on = [null_resource.wait_for_cloudinit]

  connection {
    type        = "ssh"
    host        = hcloud_server.this.ipv4_address
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      "docker swarm join --token ${var.swarm_join_token} ${var.manager_ip}:2377 2>/dev/null || echo 'Already part of swarm'",
    ]
  }
}

# ─── Node Labels (set from manager for workers) ─────────────────────
resource "null_resource" "node_labels" {
  count      = var.node_role == "worker" && length(var.node_labels) > 0 ? 1 : 0
  depends_on = [null_resource.swarm_join]

  # Labels are set from manager via SSH
  connection {
    type        = "ssh"
    host        = var.manager_public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      for key, value in var.node_labels :
      "docker node update --label-add ${key}=${value} ${var.name}"
    ]
  }
}
