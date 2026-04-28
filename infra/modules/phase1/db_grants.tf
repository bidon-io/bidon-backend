provider "postgresql" {
  host            = digitalocean_database_cluster.postgres.host
  port            = digitalocean_database_cluster.postgres.port
  username        = digitalocean_database_cluster.postgres.user
  password        = digitalocean_database_cluster.postgres.password
  database        = digitalocean_database_db.app.name
  sslmode         = "require"
  connect_timeout = 15
  superuser       = false
}

# Grant the app user full access to the public schema so migrations can create tables.
# Required on DO managed Postgres 15+ where public schema CREATE is revoked by default.
resource "postgresql_grant" "app_schema_public" {
  depends_on  = [digitalocean_database_firewall.postgres]
  database    = digitalocean_database_db.app.name
  role        = digitalocean_database_user.app.name
  schema      = "public"
  object_type = "schema"
  privileges  = ["CREATE", "USAGE"]
}

resource "postgresql_grant" "app_database" {
  depends_on  = [digitalocean_database_firewall.postgres]
  database    = digitalocean_database_db.app.name
  role        = digitalocean_database_user.app.name
  object_type = "database"
  privileges  = ["CONNECT", "CREATE"]
}
