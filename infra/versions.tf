terraform {
  required_version = ">= 1.5.0"

  # Recommended: use a remote backend for state. 
  # Example for DigitalOcean Spaces (S3 compatible):
  # backend "s3" {
  #   endpoint                    = "fra1.digitaloceanspaces.com" # match your spaces_region
  #   region                      = "us-east-1"                  # dummy for DO
  #   bucket                      = "your-terraform-state-bucket"
  #   key                         = "bidon/terraform.tfstate"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   skip_requesting_account_id  = true
  # }

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.44"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}
