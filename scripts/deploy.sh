#!/bin/bash
# Uso: ./scripts/deploy.sh [staging|prod] [image_tag]
set -euo pipefail

ENV=${1:?Especificar entorno: staging|prod}
IMAGE_TAG=${2:?Especificar image tag}
EC2_IP=$(terraform -chdir=deployments/terraform output -raw elastic_ip)
ECR_REPO=$(aws ecr describe-repositories \
    --repository-names erp-p2p-cloud \
    --query 'repositories[0].repositoryUri' \
    --output text)

echo "Desplegando $IMAGE_TAG a $ENV ($EC2_IP)..."

SSH_KEY="${SSH_KEY_PATH:-/tmp/key.pem}"

# SSH al servidor y actualizar el contenedor
ssh -o StrictHostKeyChecking=no -i "$SSH_KEY" ubuntu@"$EC2_IP" << EOF
  aws ecr get-login-password --region us-east-1 | \
      docker login --username AWS --password-stdin $ECR_REPO

  docker pull ${ECR_REPO}:${IMAGE_TAG}

  # Actualizar el tag en el service file y reiniciar
  sudo sed -i "s|${ECR_REPO}:.*|${ECR_REPO}:${IMAGE_TAG}|" /etc/systemd/system/erp-p2p-cloud.service
  sudo systemctl daemon-reload
  sudo systemctl restart erp-p2p-cloud

  # Esperar hasta que /readyz responda
  for i in \$(seq 1 12); do
    if curl -sf http://localhost:8080/readyz; then
      echo ""
      echo "Servidor saludable"
      exit 0
    fi
    echo "Esperando... intento \$i/12"
    sleep 5
  done
  echo "ERROR: El servidor no respondió en 60s"
  exit 1
EOF

echo "Deploy completado: $ENV @ $IMAGE_TAG"
