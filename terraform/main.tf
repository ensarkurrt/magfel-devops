provider "hcloud" {
  token = var.hcloud_token
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

# ─── Manager Node ────────────────────────────────────────────────────
module "manager" {
  source = "./modules/swarm-node"

  name        = "${var.cluster_name}-manager"
  server_type = var.manager.server_type
  location    = var.location
  ssh_key_id  = hcloud_ssh_key.default.id
  network_id  = hcloud_network.swarm.id
  private_ip  = var.manager.private_ip
  subnet_id   = hcloud_network_subnet.swarm.id

  node_role        = "manager"
  swarm_join_token = "" # Manager init eder, token'a ihtiyacı yok
  manager_ip       = "" # Kendisi manager

  ssh_private_key_path = var.ssh_private_key_path
}

# ─── Worker Nodes ────────────────────────────────────────────────────
module "workers" {
  source   = "./modules/swarm-node"
  for_each = var.workers

  name        = "${var.cluster_name}-${each.key}"
  server_type = each.value.server_type
  location    = var.location
  ssh_key_id  = hcloud_ssh_key.default.id
  network_id  = hcloud_network.swarm.id
  private_ip  = each.value.private_ip
  subnet_id   = hcloud_network_subnet.swarm.id

  node_role        = "worker"
  swarm_join_token = module.manager.swarm_worker_token
  manager_ip       = var.manager.private_ip
  manager_public_ip = module.manager.public_ip
  node_labels      = each.value.labels

  ssh_private_key_path = var.ssh_private_key_path
}
