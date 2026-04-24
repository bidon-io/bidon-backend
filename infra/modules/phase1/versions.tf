terraform {
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
