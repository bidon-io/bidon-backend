output "coolify_install" {
  description = "Whether Coolify was wired via cloud-init and where to find install logs on the Droplet."
  value       = module.phase1.coolify_install
}

output "droplet_ipv4" {
  description = "Public IPv4 of the Phase 1 application Droplet (Coolify / Redpanda / services host)."
  value       = module.phase1.droplet_ipv4
}

output "droplet_id" {
  description = "DigitalOcean Droplet ID."
  value       = module.phase1.droplet_id
}

output "vpc_id" {
  description = "VPC UUID shared by the Droplet and Managed Postgres for private connectivity."
  value       = module.phase1.vpc_id
}

output "postgres_cluster_id" {
  description = "Managed Postgres cluster ID."
  value       = module.phase1.postgres_cluster_id
}

output "postgres_host" {
  description = "Managed Postgres hostname (public)."
  value       = module.phase1.postgres_host
  sensitive   = true
}

output "postgres_private_host" {
  description = "Managed Postgres hostname on the private VPC network."
  value       = module.phase1.postgres_private_host
  sensitive   = true
}

output "postgres_connection_uri" {
  description = "Admin Postgres URI (rotate credentials after bootstrap; use db_app_private_uri for services)."
  value       = module.phase1.postgres_connection_uri
  sensitive   = true
}

output "postgres_private_uri" {
  description = "Admin private-network Postgres URI."
  value       = module.phase1.postgres_private_uri
  sensitive   = true
}

output "db_app_user" {
  description = "Application database username."
  value       = module.phase1.db_app_user
}

output "db_app_password" {
  description = "Application database user password."
  value       = module.phase1.db_app_password
  sensitive   = true
}

output "db_app_private_uri" {
  description = "Private-network Postgres URI for the application user. Use this in DATABASE_URL env vars."
  value       = module.phase1.db_app_private_uri
  sensitive   = true
}

output "spaces_bucket_name" {
  description = "Spaces bucket name for backups / S3-compatible tooling (null if Spaces was not provisioned)."
  value       = module.phase1.spaces_bucket_name
}

output "spaces_bucket_urn" {
  description = "URN of the Spaces bucket."
  value       = module.phase1.spaces_bucket_urn
}

output "frontend_bucket_name" {
  description = "Spaces bucket for staging frontend deployments (null if not provisioned)."
  value       = module.phase1.frontend_bucket_name
}

output "frontend_bucket_origin" {
  description = "Spaces origin hostname for nginx proxy_pass (null if not provisioned)."
  value       = module.phase1.frontend_bucket_origin
}

output "frontend_cdn_endpoint" {
  description = "CDN endpoint hostname for nginx proxy_pass or direct browser access (null if not provisioned)."
  value       = module.phase1.frontend_cdn_endpoint
}

output "next_steps" {
  description = "Human-oriented handoff for Coolify and Redpanda (not applied by Terraform)."
  value       = module.phase1.next_steps
}
