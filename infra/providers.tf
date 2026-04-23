provider "digitalocean" {
  # If var.do_token is null, the provider falls back to DIGITALOCEAN_TOKEN.
  token = var.do_token
}

provider "digitalocean" {
  alias = "spaces"
  token = var.do_token
  # Spaces credentials: set SPACES_ACCESS_KEY_ID / SPACES_SECRET_ACCESS_KEY,
  # or pass spaces_access_key / spaces_secret_key as Terraform variables (wired below).
  spaces_access_id  = var.spaces_access_key
  spaces_secret_key = var.spaces_secret_key
}
