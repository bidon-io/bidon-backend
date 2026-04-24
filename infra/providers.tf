provider "digitalocean" {
  # If var.do_token is null, the provider falls back to DIGITALOCEAN_TOKEN.
  token = var.do_token
  # Spaces credentials: set SPACES_ACCESS_KEY_ID / SPACES_SECRET_ACCESS_KEY env vars,
  # or pass spaces_access_key / spaces_secret_key as Terraform variables.
  spaces_access_id  = var.spaces_access_key
  spaces_secret_key = var.spaces_secret_key
}
