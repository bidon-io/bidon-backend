locals {
  # Map free-form environment labels to the DigitalOcean project environment enum.
  do_project_env = lookup({
    "prod"        = "Production"
    "production"  = "Production"
    "staging"     = "Staging"
    "stage"       = "Staging"
    "dev"         = "Development"
    "development" = "Development"
  }, lower(var.environment), "Development")

  droplet_entity = digitalocean_droplet.app.urn

  db_app_private_uri = "postgresql://${digitalocean_database_user.app.name}:${digitalocean_database_user.app.password}@${digitalocean_database_cluster.postgres.private_host}:${digitalocean_database_cluster.postgres.port}/${digitalocean_database_db.app.name}?sslmode=require"
}

resource "digitalocean_vpc" "main" {
  name   = "${var.name_prefix}-vpc"
  region = var.region
}

resource "digitalocean_droplet" "app" {
  name     = "${var.name_prefix}-app"
  region   = var.region
  size     = var.droplet_size
  image    = var.droplet_image
  vpc_uuid = digitalocean_vpc.main.id

  user_data = var.install_coolify ? file("${path.module}/cloud-init-coolify.yaml") : null

  backups    = var.enable_droplet_backups
  ipv6       = true
  monitoring = true
  ssh_keys   = var.ssh_key_fingerprints

  tags = [
    var.name_prefix,
    "phase1",
    "coolify-host",
  ]

  lifecycle {
    create_before_destroy = true
  }
}

resource "digitalocean_database_cluster" "postgres" {
  name       = "${var.name_prefix}-pg"
  engine     = "pg"
  version    = var.postgres_version
  size       = var.postgres_size
  region     = var.region
  node_count = 1

  private_network_uuid = digitalocean_vpc.main.id

  maintenance_window {
    day  = "sunday"
    hour = "03:00:00"
  }

  tags = [
    var.name_prefix,
    "phase1",
  ]

  lifecycle {
    prevent_destroy = true
  }
}

resource "digitalocean_database_db" "app" {
  cluster_id = digitalocean_database_cluster.postgres.id
  name       = var.postgres_db_name
}

resource "digitalocean_database_user" "app" {
  cluster_id = digitalocean_database_cluster.postgres.id
  name       = "bidon_app"
}

resource "digitalocean_database_firewall" "postgres" {
  cluster_id = digitalocean_database_cluster.postgres.id

  rule {
    type  = "droplet"
    value = digitalocean_droplet.app.id
  }
}

resource "digitalocean_firewall" "app" {
  name = "${var.name_prefix}-fw"

  droplet_ids = [digitalocean_droplet.app.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Coolify dashboard. Restrict to office/VPN CIDRs via coolify_source_cidrs before production use.
  inbound_rule {
    protocol         = "tcp"
    port_range       = "8000"
    source_addresses = length(var.coolify_source_cidrs) > 0 ? var.coolify_source_cidrs : ["0.0.0.0/0", "::/0"]
  }

  dynamic "inbound_rule" {
    for_each = length(var.ssh_admin_source_cidrs) > 0 ? [1] : []
    content {
      protocol         = "tcp"
      port_range       = "22"
      source_addresses = var.ssh_admin_source_cidrs
    }
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

resource "random_id" "spaces_suffix" {
  count       = var.enable_spaces_bucket ? 1 : 0
  byte_length = 3
}

resource "digitalocean_spaces_bucket" "backups" {
  count    = var.enable_spaces_bucket ? 1 : 0
  provider = digitalocean.spaces

  name   = "${var.name_prefix}-backups-${random_id.spaces_suffix[0].hex}"
  region = var.spaces_region
  acl    = "private"

  lifecycle {
    prevent_destroy = true
  }
}

resource "digitalocean_monitor_alert" "cpu_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet CPU is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/cpu"
  value  = 80
  window = "5m"
}

resource "digitalocean_monitor_alert" "memory_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet memory utilization is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/memory_utilization_percent"
  value  = 85
  window = "5m"
}

resource "digitalocean_monitor_alert" "disk_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet disk utilization is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/disk_utilization_percent"
  value  = 85
  window = "5m"
}

resource "digitalocean_project" "main" {
  name        = var.name_prefix
  description = "Resources for ${var.name_prefix}"
  purpose     = "Web Application"
  environment = local.do_project_env

  resources = concat(
    [
      digitalocean_droplet.app.urn,
      digitalocean_database_cluster.postgres.urn,
    ],
    var.enable_spaces_bucket ? [digitalocean_spaces_bucket.backups[0].urn] : []
  )
}
