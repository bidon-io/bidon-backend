resource "digitalocean_vpc" "main" {
  name   = "${var.name_prefix}-vpc"
  region = var.region
}
