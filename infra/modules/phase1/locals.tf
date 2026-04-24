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

  droplet_entity = tostring(digitalocean_droplet.app.id)

  db_app_private_uri = "postgresql://${digitalocean_database_user.app.name}:${digitalocean_database_user.app.password}@${digitalocean_database_cluster.postgres.private_host}:${digitalocean_database_cluster.postgres.port}/${digitalocean_database_db.app.name}?sslmode=require"
}
