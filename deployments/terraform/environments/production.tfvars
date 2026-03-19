# production.tfvars
# Production Environment — P2P Relay Server
#
# Usage:
#   terraform init -backend-config=backend-production.hcl
#   terraform apply -var-file=environments/production.tfvars
#
# Estimated cost: ~$5-8/month (1x t4g.nano + EIP + CW logs)

# ===========================================================================
# General
# ===========================================================================

environment = "production"
aws_region  = "us-east-1"

# ===========================================================================
# Networking
# ===========================================================================

# SSH access — replace with your IP
allowed_ssh_cidr = "0.0.0.0/0" # CHANGE: restrict to your IP/32

# ===========================================================================
# EC2
# ===========================================================================

instance_type = "t4g.nano" # 2 vCPU, 0.5GB RAM — sufficient for relay

# ===========================================================================
# DNS
# ===========================================================================

domain_name = "bytsers.com"
subdomain   = "relay"
