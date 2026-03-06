provider "hcloud" {
  token = var.hcloud_token
}

# ─── Locals: Flatten nodes + auto IP ─────────────────────────────────
locals {
  # Manager node'larını flat listeye çevir
  # { "infra-0" = { server_type, location, labels, ... }, "infra-1" = ... }
  manager_nodes = merge([
    for group_name, group in var.managers : {
      for i in range(group.replicas) :
      "${group_name}-${i}" => {
        group       = group_name
        server_type = group.server_type
        location    = group.location
        labels      = group.labels
        index       = i
      }
    }
  ]...)

  # Worker node'larını flat listeye çevir
  worker_nodes = merge([
    for group_name, group in var.workers : {
      for i in range(group.replicas) :
      "${group_name}-${i}" => {
        group       = group_name
        server_type = group.server_type
        location    = group.location
        labels      = group.labels
        index       = i
      }
    }
  ]...)

  # Tüm node'ları birleştir — IP ataması için sıralı liste
  all_node_keys    = concat(keys(local.manager_nodes), keys(local.worker_nodes))
  manager_node_keys = keys(local.manager_nodes)

  # Otomatik IP ataması: offset'ten başlayarak sıralı
  # manager'lar önce, worker'lar sonra
  node_ips = {
    for idx, key in local.all_node_keys :
    key => cidrhost(var.private_subnet, var.ip_offset + idx)
  }

  # İlk manager = Swarm'ı başlatan (leader)
  # Sıralama: ilk manager grubunun ilk replica'sı
  leader_key = local.manager_node_keys[0]
  leader_ip  = local.node_ips[local.leader_key]
}

# ─── SSH Key ──────────────────────────────────────────────────────────
resource "hcloud_ssh_key" "default" {
  name       = "${var.cluster_name}-key"
  public_key = file(pathexpand(var.ssh_public_key_path))
}

# ─── Private Network ─────────────────────────────────────────────────
resource "hcloud_network" "swarm" {
  name     = "${var.cluster_name}-network"
  ip_range = var.private_subnet
}

resource "hcloud_network_subnet" "swarm" {
  network_id   = hcloud_network.swarm.id
  type         = "cloud"
  network_zone = var.network_zone
  ip_range     = var.private_subnet
}

# ─── Manager Nodes ───────────────────────────────────────────────────
module "manager_nodes" {
  source   = "./modules/swarm-node"
  for_each = local.manager_nodes

  name        = "${var.cluster_name}-mgr-${each.key}"
  server_type = each.value.server_type
  location    = each.value.location
  ssh_key_id  = hcloud_ssh_key.default.id
  network_id  = hcloud_network.swarm.id
  private_ip  = local.node_ips[each.key]
  subnet_id   = hcloud_network_subnet.swarm.id

  ssh_private_key_path = var.ssh_private_key_path
}

# ─── Worker Nodes ────────────────────────────────────────────────────
module "worker_nodes" {
  source   = "./modules/swarm-node"
  for_each = local.worker_nodes

  name        = "${var.cluster_name}-wkr-${each.key}"
  server_type = each.value.server_type
  location    = each.value.location
  ssh_key_id  = hcloud_ssh_key.default.id
  network_id  = hcloud_network.swarm.id
  private_ip  = local.node_ips[each.key]
  subnet_id   = hcloud_network_subnet.swarm.id

  ssh_private_key_path = var.ssh_private_key_path
}

# ═════════════════════════════════════════════════════════════════════
# SWARM ORCHESTRATION — init, join, labels
# ═════════════════════════════════════════════════════════════════════

# ─── Step 1: Swarm Init on Leader ────────────────────────────────────
resource "null_resource" "swarm_init" {
  depends_on = [module.manager_nodes]

  connection {
    type        = "ssh"
    host        = module.manager_nodes[local.leader_key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
    timeout     = "5m"
  }

  provisioner "remote-exec" {
    inline = [
      "docker swarm init --advertise-addr ${local.leader_ip} 2>/dev/null || echo 'Swarm already initialized'",
      "docker network create --driver overlay --attachable backend 2>/dev/null || echo 'Network backend already exists'",
    ]
  }
}

# ─── Step 2: Get Swarm Tokens ────────────────────────────────────────
data "external" "swarm_tokens" {
  depends_on = [null_resource.swarm_init]

  program = ["bash", "-c", <<-EOF
    SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR"
    KEY="${pathexpand(var.ssh_private_key_path)}"
    HOST="${module.manager_nodes[local.leader_key].public_ip}"
    WORKER=$(ssh $SSH_OPTS -i "$KEY" root@"$HOST" "docker swarm join-token -q worker")
    MANAGER=$(ssh $SSH_OPTS -i "$KEY" root@"$HOST" "docker swarm join-token -q manager")
    echo "{\"worker\": \"$WORKER\", \"manager\": \"$MANAGER\"}"
  EOF
  ]
}

# ─── Step 3: Additional Managers Join ────────────────────────────────
resource "null_resource" "manager_join" {
  for_each = {
    for key, node in local.manager_nodes : key => node
    if key != local.leader_key # leader hariç
  }

  depends_on = [data.external.swarm_tokens]

  connection {
    type        = "ssh"
    host        = module.manager_nodes[each.key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      "docker swarm join --token ${data.external.swarm_tokens.result.manager} ${local.leader_ip}:2377 2>/dev/null || echo 'Already in swarm'",
    ]
  }
}

# ─── Step 4: Workers Join ────────────────────────────────────────────
resource "null_resource" "worker_join" {
  for_each = local.worker_nodes

  depends_on = [data.external.swarm_tokens]

  connection {
    type        = "ssh"
    host        = module.worker_nodes[each.key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      "docker swarm join --token ${data.external.swarm_tokens.result.worker} ${local.leader_ip}:2377 2>/dev/null || echo 'Already in swarm'",
    ]
  }
}

# ─── Step 5: Node Labels ────────────────────────────────────────────
resource "null_resource" "manager_labels" {
  for_each = {
    for key, node in local.manager_nodes : key => node
    if length(node.labels) > 0
  }

  depends_on = [null_resource.swarm_init, null_resource.manager_join]

  connection {
    type        = "ssh"
    host        = module.manager_nodes[local.leader_key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      for k, v in each.value.labels :
      "docker node update --label-add ${k}=${v} ${var.cluster_name}-mgr-${each.key}"
    ]
  }
}

resource "null_resource" "worker_labels" {
  for_each = {
    for key, node in local.worker_nodes : key => node
    if length(node.labels) > 0
  }

  depends_on = [null_resource.worker_join]

  connection {
    type        = "ssh"
    host        = module.manager_nodes[local.leader_key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  provisioner "remote-exec" {
    inline = [
      for k, v in each.value.labels :
      "docker node update --label-add ${k}=${v} ${var.cluster_name}-wkr-${each.key}"
    ]
  }
}

# ═════════════════════════════════════════════════════════════════════
# STACK DEPLOYMENT
# ═════════════════════════════════════════════════════════════════════

resource "null_resource" "deploy_stacks" {
  count = var.deploy_stacks ? 1 : 0

  depends_on = [
    null_resource.swarm_init,
    null_resource.manager_join,
    null_resource.worker_join,
    null_resource.manager_labels,
    null_resource.worker_labels,
  ]

  connection {
    type        = "ssh"
    host        = module.manager_nodes[local.leader_key].public_ip
    user        = "root"
    private_key = file(pathexpand(var.ssh_private_key_path))
  }

  # Stack dosyalarını manager'a kopyala
  provisioner "file" {
    source      = "${path.module}/../stacks/"
    destination = "/opt/stacks"
  }

  # Traefik dynamic config'i de kopyala
  provisioner "file" {
    source      = "${path.module}/../traefik/"
    destination = "/opt/traefik"
  }

  # Stack'leri sıralı deploy et
  provisioner "remote-exec" {
    inline = [
      for stack in var.stacks :
      "echo 'Deploying ${stack.name}...' && docker stack deploy -c /opt/stacks/${stack.name}/docker-compose.yml ${stack.name} --with-registry-auth 2>&1 && echo '  OK: ${stack.name}' || echo '  WARN: ${stack.name} failed'"
    ]
  }
}
