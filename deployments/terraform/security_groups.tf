resource "aws_security_group" "relay" {
  name        = "bytsers-p2p-relay"
  description = "P2P relay server security group"
  vpc_id      = aws_vpc.main.id

  # libp2p TCP
  ingress {
    description = "libp2p TCP"
    from_port   = 4001
    to_port     = 4001
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # libp2p QUIC
  ingress {
    description = "libp2p QUIC"
    from_port   = 4001
    to_port     = 4001
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SSH
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  # Health endpoint (VPC only)
  ingress {
    description = "Health check"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  # Metrics endpoint (VPC only)
  ingress {
    description = "Prometheus metrics"
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  # All egress
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "bytsers-p2p-relay" }
}
