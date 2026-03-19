# Variables
BINARY_NAME     = erp-p2p-cloud
DOCKER_IMAGE    = bytsers/erp-p2p-cloud
AWS_REGION      = us-east-1
ECR_REPO        ?= $(shell aws ecr describe-repositories --repository-names erp-p2p-cloud --query 'repositories[0].repositoryUri' --output text 2>/dev/null)
IMAGE_TAG       ?= $(shell git rev-parse --short HEAD)
GO_FLAGS        = -ldflags="-w -s -X main.version=$(IMAGE_TAG)"

.PHONY: help dev build push test test-unit test-int lint generate \
        tf-plan tf-apply tf-destroy deploy-staging deploy-prod \
        logs healthcheck gen-identity clean \
        compose-up compose-down compose-logs

help: ## Mostrar esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Iniciar servidor en modo desarrollo
	APP_ENV=dev LOG_LEVEL=debug go run ./cmd/server

build: ## Construir imagen Docker linux/arm64
	docker build --platform linux/arm64 \
		-t $(DOCKER_IMAGE):$(IMAGE_TAG) \
		-t $(DOCKER_IMAGE):latest \
		-f deployments/docker/Dockerfile .

push: ## Subir imagen a ECR
	aws ecr get-login-password --region $(AWS_REGION) | \
		docker login --username AWS --password-stdin $(ECR_REPO)
	docker tag $(DOCKER_IMAGE):$(IMAGE_TAG) $(ECR_REPO):$(IMAGE_TAG)
	docker push $(ECR_REPO):$(IMAGE_TAG)

test: ## Ejecutar todos los tests con -race
	go test ./... -race -timeout 120s -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

test-unit: ## Solo tests unitarios (rapidos, sin red)
	go test ./... -short -race -timeout 30s

test-int: ## Tests de integracion (nodos libp2p en proceso)
	go test ./... -run Integration -race -timeout 120s -v

lint: ## Ejecutar golangci-lint
	golangci-lint run ./...

generate: ## Ejecutar go generate
	go generate ./...

tf-plan: ## terraform plan (requiere AWS credentials)
	cd deployments/terraform && terraform plan

tf-apply: ## terraform apply
	cd deployments/terraform && terraform apply

tf-destroy: ## terraform destroy
	cd deployments/terraform && terraform destroy

deploy-staging: ## Desplegar a EC2 de staging
	./scripts/deploy.sh staging $(IMAGE_TAG)

deploy-prod: ## Desplegar a EC2 de produccion (requiere aprobacion)
	./scripts/deploy.sh prod $(IMAGE_TAG)

logs: ## Seguir logs desde CloudWatch
	aws logs tail /bytsers/p2p-cloud --follow --region $(AWS_REGION)

healthcheck: ## Verificar /healthz y /readyz
	@curl -sf http://localhost:8080/healthz && echo " healthz: OK"
	@curl -sf http://localhost:8080/readyz  && echo " readyz:  OK"

gen-identity: ## Generar keypair Ed25519 y guardar en Secrets Manager
	go run ./scripts/gen_identity/main.go

compose-up: ## Levantar stack observabilidad (p2p + prometheus + grafana)
	docker compose -f deployments/docker/docker-compose.yml up --build -d

compose-down: ## Detener stack observabilidad
	docker compose -f deployments/docker/docker-compose.yml down

compose-logs: ## Seguir logs del stack
	docker compose -f deployments/docker/docker-compose.yml logs -f

clean: ## Limpiar binarios y artefactos
	rm -f $(BINARY_NAME) coverage.out
	docker rmi $(DOCKER_IMAGE):$(IMAGE_TAG) 2>/dev/null || true
