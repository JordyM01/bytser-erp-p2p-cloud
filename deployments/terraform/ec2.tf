resource "tls_private_key" "relay" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "relay" {
  key_name   = "bytsers-p2p-relay-${var.environment}"
  public_key = tls_private_key.relay.public_key_openssh
}

resource "local_file" "ssh_key" {
  content         = tls_private_key.relay.private_key_pem
  filename        = "${path.module}/bytsers-p2p-relay.pem"
  file_permission = "0600"
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }
}

resource "aws_instance" "relay" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.relay.key_name
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.relay.id]
  iam_instance_profile   = aws_iam_instance_profile.relay.name

  user_data = templatefile("${path.module}/templates/user_data.sh", {
    aws_region     = var.aws_region
    ecr_repo_url   = aws_ecr_repository.relay.repository_url
    log_group_name = aws_cloudwatch_log_group.relay.name
    environment    = var.environment
  })

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  root_block_device {
    volume_size = 8
    volume_type = "gp3"
    encrypted   = true
  }

  lifecycle {
    ignore_changes = [ami, user_data]
  }

  tags = { Name = "bytsers-p2p-relay-${var.environment}" }
}
