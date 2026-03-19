variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "staging"
}

variable "instance_type" {
  description = "EC2 instance type (ARM64)"
  type        = string
  default     = "t4g.nano"
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH into the instance (e.g. YOUR_IP/32)"
  type        = string
}

variable "domain_name" {
  description = "Root domain name"
  type        = string
  default     = "bytsers.com"
}

variable "subdomain" {
  description = "Subdomain for the relay service"
  type        = string
  default     = "relay"
}
