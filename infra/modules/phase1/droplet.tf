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

  # Coolify real-time communications and terminal access.
  inbound_rule {
    protocol         = "tcp"
    port_range       = "6001"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "6002"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # SSH access. Restrict to office/VPN CIDRs via ssh_admin_source_cidrs before production use.
  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = length(var.ssh_admin_source_cidrs) > 0 ? var.ssh_admin_source_cidrs : ["0.0.0.0/0", "::/0"]
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
