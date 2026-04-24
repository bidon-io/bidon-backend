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
