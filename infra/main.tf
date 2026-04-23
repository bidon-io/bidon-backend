locals {
  name_prefix = "${var.project_name}-${var.environment}"
}

module "phase1" {
  source = "./modules/phase1"

  providers = {
    digitalocean        = digitalocean
    digitalocean.spaces = digitalocean.spaces
  }

  name_prefix = local.name_prefix
  environment = var.environment
  region      = var.region

  droplet_size           = var.droplet_size
  droplet_image          = var.droplet_image
  install_coolify        = var.install_coolify
  enable_droplet_backups = var.enable_weekly_droplet_backups
  ssh_key_fingerprints   = var.ssh_key_fingerprints
  ssh_admin_source_cidrs = var.ssh_admin_source_cidrs
  coolify_source_cidrs   = var.coolify_source_cidrs

  postgres_version = var.postgres_version
  postgres_size    = var.postgres_size
  postgres_db_name = var.postgres_db_name

  spaces_region        = var.spaces_region
  enable_spaces_bucket = var.enable_spaces_bucket

  monitor_alert_email = var.monitor_alert_email
}
