resource "random_id" "spaces_suffix" {
  count       = var.enable_spaces_bucket ? 1 : 0
  byte_length = 3
}

resource "digitalocean_spaces_bucket" "backups" {
  count = var.enable_spaces_bucket ? 1 : 0

  name   = "${var.name_prefix}-backups-${random_id.spaces_suffix[0].hex}"
  region = var.spaces_region
  acl    = "private"

  lifecycle {
    # To enable: set to true and re-apply. Variables are not allowed in lifecycle blocks.
    prevent_destroy = false
  }
}

# Staging frontend bucket.
# CI deploys write to two prefixes on every build:
#   current/              — what nginx serves; overwritten on every deploy
#   releases/<git-sha>/   — immutable snapshot; used for rollback via:
#                           aws s3 sync s3://<bucket>/releases/<sha>/ s3://<bucket>/current/ --delete

resource "random_id" "frontend_suffix" {
  count       = var.enable_frontend_bucket ? 1 : 0
  byte_length = 3
}

resource "digitalocean_spaces_bucket" "frontend" {
  count = var.enable_frontend_bucket ? 1 : 0

  name   = "${var.name_prefix}-frontend-${random_id.frontend_suffix[0].hex}"
  region = var.spaces_region
  acl    = "public-read"

  lifecycle {
    prevent_destroy = false
  }
}

resource "digitalocean_cdn" "frontend" {
  count  = var.enable_frontend_bucket ? 1 : 0
  origin = digitalocean_spaces_bucket.frontend[0].bucket_domain_name
}
