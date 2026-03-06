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
  ssh_keys     = [var.ssh_key_id]
  firewall_ids = [var.firewall_id]
  user_data    = data.cloudinit_config.this.rendered

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  lifecycle {
    ignore_changes = [user_data]
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
      "cloud-init status --wait",
      "docker --version",
    ]
  }
}
