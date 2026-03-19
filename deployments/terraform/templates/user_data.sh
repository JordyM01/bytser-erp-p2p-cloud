#!/bin/bash
set -euo pipefail

# Wait for cloud-init network
until ping -c1 archive.ubuntu.com &>/dev/null; do sleep 2; done

# Install Docker and AWS CLI
apt-get update -y
apt-get install -y docker.io awscli
systemctl enable docker
systemctl start docker
usermod -aG docker ubuntu

# ECR login
aws ecr get-login-password --region ${aws_region} | \
    docker login --username AWS --password-stdin ${ecr_repo_url}

# Environment file
cat > /opt/erp-p2p-cloud.env <<ENVEOF
APP_ENV=${environment}
AWS_DEFAULT_REGION=${aws_region}
ECR_REPO=${ecr_repo_url}
ENVEOF

# Systemd service
cat > /etc/systemd/system/erp-p2p-cloud.service <<SVCEOF
[Unit]
Description=ERP P2P Cloud Relay
After=docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
RestartSec=10
EnvironmentFile=/opt/erp-p2p-cloud.env
ExecStartPre=-/usr/bin/docker rm -f erp-p2p-cloud
ExecStart=/usr/bin/docker run --rm --name erp-p2p-cloud \
    --env-file /opt/erp-p2p-cloud.env \
    -p 4001:4001/tcp \
    -p 4001:4001/udp \
    -p 8080:8080 \
    -p 9090:9090 \
    --log-driver=awslogs \
    --log-opt awslogs-region=${aws_region} \
    --log-opt awslogs-group=${log_group_name} \
    --log-opt awslogs-stream=relay \
    --log-opt awslogs-create-group=true \
    ${ecr_repo_url}:latest
ExecStop=/usr/bin/docker stop erp-p2p-cloud

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable erp-p2p-cloud
# Service will fail until an image is pushed — that's expected
systemctl start erp-p2p-cloud || true
