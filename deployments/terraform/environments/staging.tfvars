# staging.tfvars
# Staging Environment — P2P Relay Server
#
# Usage:
#   terraform init -backend-config=backend-staging.hcl
#   terraform apply -var-file=environments/staging.tfvars
#
# Estimated cost: ~$5-8/month (1x t4g.nano + EIP + CW logs)

# ===========================================================================
# General
# ===========================================================================

environment = "staging"
aws_region  = "us-east-1"

# ===========================================================================
# Networking
# ===========================================================================

# SSH access — replace with your IP
allowed_ssh_cidr = "0.0.0.0/0" # CHANGE: restrict to your IP/32

# ===========================================================================
# EC2
# ===========================================================================

instance_type = "t4g.nano"

# ===========================================================================
# DNS
# ===========================================================================

domain_name = "bytsers.com"
subdomain   = "relay-staging"
