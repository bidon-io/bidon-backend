variable "do_token" {
  type        = string
  sensitive   = true
  description = "DigitalOcean personal access token. If null, the provider uses DIGITALOCEAN_TOKEN from the environment."
  default     = null
  nullable    = true
}

variable "spaces_access_key" {
  type        = string
  sensitive   = true
  description = "Spaces access key. If null, use SPACES_ACCESS_KEY_ID (DigitalOcean Terraform provider convention)."
  default     = null
  nullable    = true
}

variable "spaces_secret_key" {
  type        = string
  sensitive   = true
  description = "Spaces secret key. If null, use SPACES_SECRET_ACCESS_KEY."
  default     = null
  nullable    = true
}

variable "project_name" {
  type        = string
  description = "Short name prefix for resources (letters, digits, hyphens)."
  default     = "bidon"
}

variable "environment" {
  type        = string
  description = "Deployment label (e.g. prod, staging)."
  default     = "prod"
}

variable "region" {
  type        = string
  description = "DigitalOcean region slug for the Droplet, VPC, and Managed Postgres (e.g. fra1, nyc3, ams3)."
  default     = "fra1"
}

variable "spaces_region" {
  type        = string
  description = "Region slug for the Spaces bucket (must be a Spaces-enabled region, e.g. ams3 for fra1 workloads)."
  default     = "ams3"
}

variable "droplet_size" {
  type        = string
  description = "Droplet size slug. Proposal baseline: 4 vCPU / 8 GB (Standard: s-4vcpu-8gb; Premium Intel GP: g-4vcpu-8gb where available)."
  default     = "s-4vcpu-8gb"
}

variable "droplet_image" {
  type        = string
  description = "OS image slug for the application Droplet (Coolify host)."
  default     = "ubuntu-22-04-x64"
}

variable "install_coolify" {
  type        = bool
  description = "Run the official Coolify install via Droplet cloud-init on first boot. Disable to provision the VM only and install Coolify manually later."
  default     = true
}

variable "postgres_version" {
  type        = string
  description = "Managed Postgres major version."
  default     = "16"
}

variable "postgres_size" {
  type        = string
  description = "Managed Postgres node size slug (Basic tier example: db-s-1vcpu-1gb)."
  default     = "db-s-1vcpu-1gb"
}

variable "postgres_db_name" {
  type        = string
  description = "Application database name created inside the cluster."
  default     = "bidon"
}

variable "ssh_key_fingerprints" {
  type        = list(string)
  description = "SSH public key fingerprints already registered in DigitalOcean (https://cloud.digitalocean.com/account/security). Empty = rely on console recovery / add keys later."
  default     = []
}

variable "ssh_admin_source_cidrs" {
  type        = list(string)
  description = "CIDR blocks allowed to reach TCP/22 on the Droplet. Empty = open to the world. Restrict to office/VPN CIDRs before production use."
  default     = []
}

variable "coolify_source_cidrs" {
  type        = list(string)
  description = "CIDR blocks allowed to reach the Coolify dashboard on TCP/8000. Empty = open to the world. Restrict to office/VPN CIDRs before production use."
  default     = []
}

variable "enable_weekly_droplet_backups" {
  type        = bool
  description = "Enable DigitalOcean weekly Droplet backups (extra cost, matches proposal)."
  default     = true
}

variable "enable_spaces_bucket" {
  type        = bool
  description = "Create a private Spaces bucket for Postgres backup archives and future object storage. Requires Spaces API credentials (env vars or Terraform variables)."
  default     = true
}

variable "monitor_alert_email" {
  type        = string
  description = "Email address for DigitalOcean Insights alerts on the Droplet (CPU/memory/disk). If null, monitor alerts are not created."
  default     = null
  nullable    = true
}
