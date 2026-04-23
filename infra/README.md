# Infrastructure - Phase 1

This directory contains Terraform configurations for provisioning the Phase 1 infrastructure on DigitalOcean.

## Architecture

- **VPC**: A dedicated Virtual Private Cloud for all resources.
- **Application Droplet**: A single Ubuntu Droplet running:
  - **Coolify**: For application deployment and management.
  - **Redpanda**: Managed via Docker on the same host (Phase 1).
- **Managed Postgres**: A DigitalOcean Managed Database cluster (PostgreSQL 16, single node).
- **Spaces Bucket**: S3-compatible storage for database backups and other object storage needs.
- **Firewall**: Restricts inbound traffic to HTTP (80), HTTPS (443), and Coolify (8000). SSH (22) is disabled by default and can be restricted to specific CIDR blocks.
- **Monitoring**: DigitalOcean Insights alerts for CPU (>80%), Memory (>85%), and Disk (>85%) utilization.

## Prerequisites

1. **Terraform**: Version 1.5.0 or later.
2. **DigitalOcean Token**: A Personal Access Token with read/write access.
3. **Spaces Access Keys**: Access ID and Secret Key for Spaces (S3) API (only needed if `enable_spaces_bucket = true`).
4. **SSH Keys**: Fingerprints of SSH keys already registered in your DigitalOcean account.

## Usage

1. **Initialize**:
   ```bash
   terraform init
   ```

2. **Configure**:
   Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in your values.
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

3. **Plan**:
   ```bash
   terraform plan
   ```

4. **Apply**:
   ```bash
   terraform apply
   ```

## Key Outputs

```bash
# Public IP (Coolify dashboard, DNS target)
terraform output droplet_ipv4

# Application DB connection URI (use in DATABASE_URL env vars)
terraform output -raw db_app_private_uri

# Spaces bucket name (for backup tooling)
terraform output spaces_bucket_name
```

## Coolify Configuration

Terraform provisions the infrastructure only. Configuring Bidon applications inside Coolify is a separate step handled by the `bidon-coolify` CLI (`cmd/bidon-coolify/`).

### Flow

```
terraform apply          # 1. Provision infra (Droplet, Postgres, Spaces)
     ↓
cloud-init runs          # 2. Coolify installs automatically (~3-5 min after Droplet boots)
     ↓
http://<ip>:8000         # 3. Manual: open Coolify, create admin user, generate API key
     ↓
bidon-coolify            # 4. Automated: configure GitHub App, create apps, set env vars
```

### Step 3 — Initial Coolify setup (manual, one-time)

Once the Droplet is up, Coolify installs via cloud-init. Check progress on the host:
```bash
ssh root@$(terraform output -raw droplet_ipv4)
tail -f /var/log/coolify-install.log
```

Then open `http://<droplet_ip>:8000`, create the admin user, and generate an API key under
**Profile → API Tokens**.

### Step 4 — Configure Bidon apps (bidon-coolify)

```bash
export COOLIFY_BASE_URL="http://$(terraform output -raw droplet_ipv4):8000"
export COOLIFY_API_KEY="<your-api-key>"

# Register GitHub App integration
go run ./cmd/bidon-coolify configure-github-repo \
  --name bidon-gh-app \
  --organization bidon-io \
  --app-id <app_id> \
  --installation-id <installation_id> \
  --client-id <client_id> \
  --client-secret <client_secret> \
  --webhook-secret <webhook_secret> \
  --private-key-uuid <coolify_private_key_uuid>

# Create application (repeat for bidon-sdkapi etc.)
go run ./cmd/bidon-coolify create-app \
  --project-uuid <project_uuid> \
  --server-uuid <server_uuid> \
  --github-app-uuid <github_app_uuid> \
  --name bidon-admin \
  --git-repository https://github.com/bidon-io/bidon-backend \
  --image-name ghcr.io/bidon-io/bidon-admin \
  --image-tag latest \
  --ports-exposes 1323 \
  --health-check-path /health \
  --health-check-port 1323

# Set environment variables
go run ./cmd/bidon-coolify configure-app-env \
  --app-uuid <application_uuid> \
  --env DATABASE_URL="$(terraform output -raw db_app_private_uri)" \
  --env-file .env.coolify
```

See `cmd/bidon-coolify/README.md` for full flag reference.

## Security Hardening

After initial setup, restrict the Coolify dashboard to your office/VPN CIDRs:

```hcl
# terraform.tfvars
coolify_source_cidrs = ["203.0.113.10/32"]
```

Then re-apply:
```bash
terraform apply
```

## State Management

By default this project uses local state. For production, configure a remote backend (e.g., a DigitalOcean Spaces bucket) to prevent state loss. See the commented `backend "s3"` block in `versions.tf` for the configuration template.
