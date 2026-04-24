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
