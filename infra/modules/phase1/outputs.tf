output "droplet_ipv4" {
  value = digitalocean_droplet.app.ipv4_address
}

output "droplet_id" {
  value = digitalocean_droplet.app.id
}

output "vpc_id" {
  value = digitalocean_vpc.main.id
}

output "postgres_cluster_id" {
  value = digitalocean_database_cluster.postgres.id
}

output "postgres_host" {
  value     = digitalocean_database_cluster.postgres.host
  sensitive = true
}

output "postgres_private_host" {
  value     = digitalocean_database_cluster.postgres.private_host
  sensitive = true
}

output "postgres_connection_uri" {
  description = "Admin connection URI (rotate credentials after bootstrap; use app URI for services)."
  value       = digitalocean_database_cluster.postgres.uri
  sensitive   = true
}

output "postgres_private_uri" {
  description = "Admin private-network URI."
  value       = digitalocean_database_cluster.postgres.private_uri
  sensitive   = true
}

output "db_app_user" {
  description = "Application database username (least-privilege; use this for service config)."
  value       = digitalocean_database_user.app.name
}

output "db_app_password" {
  description = "Application database user password."
  value       = digitalocean_database_user.app.password
  sensitive   = true
}

output "db_app_private_uri" {
  description = "Private-network Postgres URI for the application user. Use this in service DATABASE_URL env vars."
  value       = local.db_app_private_uri
  sensitive   = true
}

output "spaces_bucket_name" {
  value = try(digitalocean_spaces_bucket.backups[0].name, null)
}

output "spaces_bucket_urn" {
  value = try(digitalocean_spaces_bucket.backups[0].urn, null)
}

output "frontend_bucket_name" {
  description = "Spaces bucket name for staging frontend deployments (null if not provisioned)."
  value       = try(digitalocean_spaces_bucket.frontend[0].name, null)
}

output "frontend_bucket_origin" {
  description = "Spaces origin hostname — use in nginx proxy_pass (e.g. https://<origin>/current/)."
  value       = try(digitalocean_spaces_bucket.frontend[0].bucket_domain_name, null)
}

output "frontend_cdn_endpoint" {
  description = "CDN endpoint hostname — cached alternative to the Spaces origin for nginx proxy_pass."
  value       = try(digitalocean_cdn.frontend[0].endpoint, null)
}

output "coolify_install" {
  description = "How Coolify was configured on the Droplet."
  value       = var.install_coolify ? "cloud-init first boot: https://cdn.coollabs.io/coolify/install.sh (log: /var/log/coolify-install.log on the server)" : "disabled in Terraform; install manually after SSH"
}

output "next_steps" {
  value = <<-EOT
    Phase 1 (proposal): Coolify and Redpanda run on the Droplet; Managed Postgres and Spaces are cloud services.

    1) SSH: add your public key in DigitalOcean, set ssh_key_fingerprints / ssh_admin_source_cidrs in Terraform, apply again if you locked SSH down.
    2) Coolify: ${var.install_coolify ? "installed automatically on first boot via cloud-init; allow several minutes after the Droplet is up, then check http://${digitalocean_droplet.app.ipv4_address}:8000 (see /var/log/coolify-install.log on the host)" : "install manually with the official script when you have SSH"} — point DNS A/AAAA records at ${digitalocean_droplet.app.ipv4_address}${digitalocean_droplet.app.ipv6_address != "" ? " and the Droplet IPv6 address" : ""}.
    3) Configure Bidon services via bidon-coolify (see cmd/bidon-coolify/README.md and infra/README.md).
    4) Configure application services with db_app_private_uri (terraform output -raw db_app_private_uri).
    5) Use the Spaces bucket (${try(digitalocean_spaces_bucket.backups[0].name, "disabled in Terraform")}) for encrypted backup archives via any S3-compatible tool.
    6) Harden: set coolify_source_cidrs to your office/VPN CIDR blocks and re-apply to restrict port 8000.
  EOT
}
