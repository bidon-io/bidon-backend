resource "random_id" "postgres_suffix" {
  byte_length = 3
}

resource "digitalocean_database_cluster" "postgres" {
  name       = "${var.name_prefix}-pg-${random_id.postgres_suffix.hex}"
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
    # To enable: set to true and re-apply. Variables are not allowed in lifecycle blocks.
    prevent_destroy = false
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

  dynamic "rule" {
    for_each = var.terraform_runner_ip != null ? [var.terraform_runner_ip] : []
    content {
      type  = "ip_addr"
      value = rule.value
    }
  }
}
