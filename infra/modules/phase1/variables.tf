variable "name_prefix" {
  type        = string
  description = "Prefix used for resource names."
}

variable "environment" {
  type        = string
  description = "Deployment environment label (prod, staging, dev). Used for naming and mapped to a DigitalOcean project environment enum internally."
}

variable "region" {
  type        = string
  description = "DigitalOcean region slug for the Droplet, VPC, and Managed Postgres (e.g. fra1, nyc3, ams3)."
}

variable "droplet_size" {
  type        = string
  description = "Droplet size slug (e.g. s-4vcpu-8gb)."
}

variable "droplet_image" {
  type        = string
  description = "OS image slug for the Droplet (e.g. ubuntu-22-04-x64)."
}

variable "enable_droplet_backups" {
  type        = bool
  description = "Enable DigitalOcean weekly Droplet backups."
}

variable "install_coolify" {
  type        = bool
  description = "If true, pass cloud-init user_data so the Droplet runs the official Coolify install script on first boot only. Changing this after create can replace the Droplet."
  default     = true
}

variable "ssh_key_fingerprints" {
  type        = list(string)
  description = "SSH public key fingerprints already registered in DigitalOcean. Empty = no SSH keys injected."
}

variable "ssh_admin_source_cidrs" {
  type        = list(string)
  description = "CIDRs allowed to SSH (TCP/22). Empty = open to the world. Restrict to office/VPN CIDRs before production use."
}

variable "coolify_source_cidrs" {
  type        = list(string)
  description = "CIDRs allowed to reach the Coolify dashboard (TCP/8000). Empty = open to the world (safe for initial setup; restrict to office/VPN CIDRs before production use)."
  default     = []
}

variable "postgres_version" {
  type        = string
  description = "Managed Postgres major version."
}

variable "postgres_size" {
  type        = string
  description = "Managed Postgres node size slug (e.g. db-s-1vcpu-1gb)."
}

variable "postgres_db_name" {
  type        = string
  description = "Application database name created inside the cluster."
}

variable "spaces_region" {
  type        = string
  description = "Region slug for the Spaces bucket (must be a Spaces-enabled region, e.g. ams3)."
}

variable "enable_spaces_bucket" {
  type        = bool
  description = "Whether to create the Spaces bucket (requires Spaces API credentials on the aliased provider)."
}

variable "monitor_alert_email" {
  type        = string
  description = "If set, creates Droplet monitor alerts for CPU, memory, and disk utilization."
  default     = null
  nullable    = true
}

variable "terraform_runner_ip" {
  type        = string
  description = "Public IP of the machine running Terraform. When set, temporarily opens the managed Postgres firewall to that IP so the postgresql provider can apply schema grants. Get it with: curl -s ifconfig.me"
  default     = null
  nullable    = true
}
