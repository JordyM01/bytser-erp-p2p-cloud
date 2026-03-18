# Bytser ERP-P2P-CLOUD — Roadmap de Desarrollo

**Versión:** 1.0  
**Estado:** ACTIVO  
**Fecha:** Marzo 2026  
**Arquitectura de referencia:** `ARQUITECTURA_ERP_P2P_CLOUD.md` v1.1 ✅ APROBADA  
**Dominio:** `relay.bytsers.com` — wildcard ACM `*.bytsers.com` ya provisionado  
**Stack:** Go 1.22+ · go-libp2p · zerolog · Prometheus · Terraform · GitHub Actions  

---

## Índice

- [Resumen Ejecutivo](#resumen-ejecutivo)
- [Mapa de Fases](#mapa-de-fases)
- [Fase 0 — Fundamentos del Repositorio](#fase-0--fundamentos-del-repositorio-semana-1)
- [Fase 1 — Identidad y Transporte P2P](#fase-1--identidad-y-transporte-p2p-semana-2)
- [Fase 2 — DHT, AutoNAT y Hole Punching](#fase-2--dht-autonat-y-hole-punching-semanas-3-4)
- [Fase 3 — Circuit Relay v2](#fase-3--circuit-relay-v2-semana-5)
- [Fase 4 — Observabilidad Completa](#fase-4--observabilidad-completa-semana-6)
- [Fase 5 — Infraestructura AWS y IaC](#fase-5--infraestructura-aws-y-iac-semana-7)
- [Fase 6 — CI/CD y Pipeline de Despliegue](#fase-6--cicd-y-pipeline-de-despliegue-semana-8)
- [Fase 7 — Hardening, Tests E2E y Go-Live](#fase-7--hardening-tests-e2e-y-go-live-semana-9)
- [Criterios de Éxito Globales](#criterios-de-éxito-globales)
- [Registro de Decisiones Técnicas Pendientes](#registro-de-decisiones-técnicas-pendientes)

---

## Resumen Ejecutivo

El proyecto se divide en **7 fases a lo largo de 9 semanas**. Las fases son secuenciales con pequeños solapamientos en las últimas semanas. El criterio de "done" de cada paso es: **código commiteado + tests pasando + CI verde**.

```
Sem 1   Sem 2   Sem 3   Sem 4   Sem 5   Sem 6   Sem 7   Sem 8   Sem 9
──────  ──────  ──────  ──────  ──────  ──────  ──────  ──────  ──────
Fase 0  Fase 1  ─ Fase 2 ────  Fase 3  Fase 4  Fase 5  Fase 6  Fase 7
Repos   Transp  DHT AutoNAT    Relay   Observ  AWS IaC CI/CD   GoLive
        Ident   DCUtR
```

**Entregable final:** Servidor `relay.bytsers.com:4001` en producción, procesando conexiones P2P reales de nodos ERP-CORE, con monitoreo completo en Grafana, pipeline CI/CD automatizado y documentación operativa.

---

## Mapa de Fases

| Fase | Nombre | Semana | Entregable Clave |
|---|---|---|---|
| 0 | Fundamentos del Repositorio | 1 | Repo inicializado, CI base verde, estructura de directorios |
| 1 | Identidad y Transporte P2P | 2 | Host go-libp2p arrancando y escuchando en puertos TCP+QUIC |
| 2 | DHT, AutoNAT y Hole Punching | 3–4 | Nodos descubriéndose y perforando NAT en tests de integración |
| 3 | Circuit Relay v2 | 5 | Relay funcional con límites configurados, tests de respaldo |
| 4 | Observabilidad Completa | 6 | /metrics, /healthz, /readyz, logs estructurados, Grafana local |
| 5 | Infraestructura AWS y IaC | 7 | Terraform apply en staging, servidor corriendo en EC2 real |
| 6 | CI/CD y Pipeline de Despliegue | 8 | Deploy automático a staging, deploy manual a producción |
| 7 | Hardening, Tests E2E y Go-Live | 9 | Producción activa, dominio configurado, runbook completo |

---

## Fase 0 — Fundamentos del Repositorio (Semana 1)

**Objetivo:** Tener la base técnica del repositorio lista para que el desarrollo de las fases siguientes fluya sin fricciones. Todo el scaffolding, convenciones y herramientas de desarrollo configuradas desde el primer día.

**Criterio de completitud:** `make dev` arranca sin errores. `make lint` pasa. `make test` pasa (aunque no haya tests aún). CI verde en el primer PR.

---

### Paso 0.1 — Inicializar Repositorio

- [ ] Crear repositorio `bytsers/erp-p2p-cloud` en GitHub
- [ ] Configurar branch protection en `main`:
  - Requerir PR con al menos 1 reviewer aprobado
  - Requerir que todos los checks de CI pasen antes de merge
  - No permitir push directo a `main`
- [ ] Crear branches base: `main` (producción), `develop` (integración)
- [ ] Agregar `README.md` con descripción del proyecto, propósito y cómo correr localmente
- [ ] Agregar `LICENSE` (privado/propietario)
- [ ] Agregar `.gitignore` para Go (binarios, `*.env`, `*.pem`, carpeta `terraform/.terraform/`)

---

### Paso 0.2 — Módulo Go y Estructura de Directorios

- [ ] Inicializar módulo Go: `go mod init github.com/bytsers/erp-p2p-cloud`
- [ ] Crear la estructura de directorios completa según la arquitectura:

  ```
  cmd/server/
  internal/node/
  internal/dht/
  internal/relay/
  internal/autonat/
  internal/metrics/
  internal/health/
  internal/config/
  deployments/docker/
  deployments/terraform/
  scripts/
  config/
  .github/workflows/
  ```

- [ ] Crear archivos `.go` vacíos con el `package` correcto en cada directorio (para que el módulo sea válido)
- [ ] Verificar: `go build ./...` compila sin errores

---

### Paso 0.3 — Dependencias Base

- [ ] Agregar dependencias al `go.mod`:

  ```bash
  go get github.com/libp2p/go-libp2p@latest
  go get github.com/libp2p/go-libp2p-kad-dht@latest
  go get github.com/rs/zerolog@latest
  go get github.com/prometheus/client_golang@latest
  go get github.com/spf13/viper@latest
  go get github.com/google/uuid@latest
  go get github.com/aws/aws-sdk-go-v2/config@latest
  go get github.com/aws/aws-sdk-go-v2/service/secretsmanager@latest
  go get github.com/stretchr/testify@latest

  # Solo dev
  go get github.com/stretchr/mock@latest
  ```

- [ ] Ejecutar `go mod tidy`
- [ ] Verificar que `go mod vendor` (o caché de módulos) funciona correctamente

---

### Paso 0.4 — Configuración del Linter

- [ ] Instalar `golangci-lint` v1.57+
- [ ] Crear `.golangci.yml` con las siguientes reglas habilitadas:

  ```yaml
  linters:
    enable:
      - errcheck       # verificar errores no manejados
      - gosimple       # simplificaciones de código
      - govet          # análisis estático del compilador
      - ineffassign    # asignaciones ineficientes
      - staticcheck    # análisis estático avanzado
      - unused         # código no usado
      - gofmt          # formato estándar Go
      - goimports      # imports ordenados
      - revive         # reemplaza golint
      - gocritic       # sugerencias de mejora
      - misspell       # errores tipográficos en comentarios
      - godot          # punto final en comentarios
      - noctx          # detectar requests HTTP sin contexto
      - bodyclose      # verificar cierre de response bodies
      - nilnil         # evitar (nil, nil) returns
  linters-settings:
    revive:
      rules:
        - name: exported
    gocritic:
      enabled-tags:
        - diagnostic
        - performance
  issues:
    exclude-rules:
      - path: "_test.go"
        linters: [errcheck]
  ```

- [ ] Verificar: `golangci-lint run ./...` pasa (con código vacío)

---

### Paso 0.5 — Makefile

- [ ] Crear `Makefile` con todos los targets definidos en la arquitectura:

  ```makefile
  # Variables
  BINARY_NAME     = erp-p2p-cloud
  DOCKER_IMAGE    = bytsers/erp-p2p-cloud
  AWS_REGION      = us-east-1
  ECR_REPO        ?= $(shell aws ecr describe-repositories --repository-names erp-p2p-cloud --query 'repositories[0].repositoryUri' --output text 2>/dev/null)
  IMAGE_TAG       ?= $(shell git rev-parse --short HEAD)
  GO_FLAGS        = -ldflags="-w -s -X main.version=$(IMAGE_TAG)"

  .PHONY: help dev build push test test-unit test-int lint \
          tf-plan tf-apply deploy-staging deploy-prod \
          logs healthcheck gen-identity clean

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

  test-unit: ## Solo tests unitarios (rápidos, sin red)
      go test ./... -short -race -timeout 30s

  test-int: ## Tests de integración (nodos libp2p en proceso)
      go test ./... -run Integration -race -timeout 120s -v

  lint: ## Ejecutar golangci-lint
      golangci-lint run ./...

  generate: ## Ejecutar go generate (si aplica en el futuro)
      go generate ./...

  tf-plan: ## terraform plan (requiere AWS credentials)
      cd deployments/terraform && terraform plan

  tf-apply: ## terraform apply
      cd deployments/terraform && terraform apply

  tf-destroy: ## terraform destroy (¡CUIDADO!)
      cd deployments/terraform && terraform destroy

  deploy-staging: ## Desplegar a EC2 de staging
      ./scripts/deploy.sh staging $(IMAGE_TAG)

  deploy-prod: ## Desplegar a EC2 de producción (requiere aprobación)
      ./scripts/deploy.sh prod $(IMAGE_TAG)

  logs: ## Seguir logs desde CloudWatch
      aws logs tail /bytsers/p2p-cloud --follow --region $(AWS_REGION)

  healthcheck: ## Verificar /healthz y /readyz
      @curl -sf http://localhost:8080/healthz && echo "healthz: OK"
      @curl -sf http://localhost:8080/readyz  && echo "readyz:  OK"

  gen-identity: ## Generar keypair Ed25519 y guardar en Secrets Manager (solo una vez)
      go run ./scripts/gen_identity/main.go

  clean: ## Limpiar binarios y artefactos
      rm -f $(BINARY_NAME) coverage.out
      docker rmi $(DOCKER_IMAGE):$(IMAGE_TAG) 2>/dev/null || true
  ```

- [ ] Verificar: `make help` imprime todos los targets correctamente

---

### Paso 0.6 — Configuración Base (Viper)

- [ ] Implementar `internal/config/config.go` con el struct completo definido en la arquitectura
- [ ] Implementar `LoadConfig(env string) (*Config, error)` que:
  1. Lee el archivo `config/config.{env}.yaml`
  2. Permite override de cualquier campo vía variable de entorno
  3. Retorna error descriptivo si falta algún campo requerido
- [ ] Crear los tres archivos YAML:
  - `config/config.dev.yaml` — puertos locales, log level debug, Secrets Manager deshabilitado (usa keypair generado localmente)
  - `config/config.staging.yaml` — usa Secrets Manager real, log level debug
  - `config/config.prod.yaml` — usa Secrets Manager real, log level info, límites de producción
- [ ] Tests unitarios para `LoadConfig`:
  - Carga correcta de valores desde YAML
  - Override correcto desde variable de entorno
  - Error cuando falta campo requerido
  - Defaults aplicados correctamente

**Criterio:** `make test-unit` pasa con cobertura > 85% en `internal/config/`

---

### Paso 0.7 — Health Handlers (Stub)

- [ ] Implementar `internal/health/handler.go`:

  ```go
  // /healthz — siempre 200 si el proceso está vivo
  func HandleHealthz(w http.ResponseWriter, r *http.Request)

  // /readyz — comprueba que el sistema está listo
  // En este paso: siempre 200 (se actualizará en Fase 1 con checks reales)
  func HandleReadyz(checks ...ReadinessCheck) http.HandlerFunc
  ```

- [ ] Tests unitarios para ambos handlers
- [ ] Implementar `cmd/server/main.go` mínimo:
  - Carga config
  - Inicia servidor HTTP en `:8080` con `/healthz` y `/readyz`
  - Maneja `SIGTERM`/`SIGINT` con `os/signal`
  - Log de inicio: `"ERP-P2P-CLOUD iniciando... versión: X env: Y"`
- [ ] Verificar: `make dev` arranca, `make healthcheck` retorna OK

---

### Paso 0.8 — CI Base (GitHub Actions)

- [ ] Crear `.github/workflows/ci.yml`:

  ```yaml
  name: CI
  on:
    pull_request:
      branches: [main, develop]
  jobs:
    lint:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version: '1.22' }
        - uses: golangci/golangci-lint-action@v4

    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version: '1.22' }
        - run: go test ./... -race -coverprofile=coverage.out -timeout 120s
        - run: |
            COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
            echo "Cobertura total: ${COVERAGE}%"
            awk -v cov="$COVERAGE" 'BEGIN { if (cov+0 < 80) { print "ERROR: Cobertura " cov "% < 80% requerido"; exit 1 } }'

    security:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version: '1.22' }
        - run: go install golang.org/x/vuln/cmd/govulncheck@latest
        - run: govulncheck ./...
  ```

- [ ] Abrir un PR de prueba para verificar que el CI corre y pasa

---

**✅ Entregables Fase 0:**

- Repositorio inicializado con estructura completa
- `make dev` arranca el servidor con health checks respondiendo
- CI verde en GitHub Actions (lint + test + security)
- Config tipada y testeada con Viper

---

## Fase 1 — Identidad y Transporte P2P (Semana 2)

**Objetivo:** El servidor puede generar/cargar su identidad Ed25519, construir un host go-libp2p funcional y escuchar conexiones entrantes en TCP y QUIC. Es el núcleo sobre el que todo lo demás se construye.

**Criterio de completitud:** El servidor arranca, imprime su PeerID en los logs, y acepta conexiones de un nodo de test go-libp2p en el mismo proceso.

---

### Paso 1.1 — Identidad Ed25519

- [ ] Implementar `internal/node/identity.go`:

  ```go
  type NodeIdentity struct {
      PrivKey  crypto.PrivKey
      PubKey   crypto.PubKey
      PeerID   peer.ID
  }

  // LoadFromSecretsManager carga el keypair desde AWS Secrets Manager.
  // El secreto se almacena como JSON: {"private_key_b64": "...", "public_key_b64": "..."}
  func LoadFromSecretsManager(ctx context.Context, secretName string, client SecretsClient) (*NodeIdentity, error)

  // GenerateAndSave genera un nuevo keypair Ed25519, lo guarda en Secrets Manager
  // y retorna la identidad. Solo se llama una vez (make gen-identity).
  func GenerateAndSave(ctx context.Context, secretName string, client SecretsClient) (*NodeIdentity, error)

  // LoadOrGenerateLocal para desarrollo: carga de archivo local o genera nuevo.
  // Solo se usa cuando APP_ENV=dev (sin Secrets Manager).
  func LoadOrGenerateLocal(keyFilePath string) (*NodeIdentity, error)
  ```

- [ ] Implementar `scripts/gen_identity/main.go`:
  - Genera keypair Ed25519
  - Imprime el PeerID resultante (para copiar a la config de ERP-CORE)
  - Guarda el keypair en AWS Secrets Manager en el secret `bytsers/p2p-relay/identity`
  - Log de confirmación con el PeerID generado

- [ ] Tests unitarios para `identity.go`:
  - `LoadOrGenerateLocal`: genera si no existe, carga si ya existe, PeerID es el mismo entre cargas
  - `LoadFromSecretsManager`: mock del cliente AWS, verifica parsing correcto del JSON
  - Verificar que el PeerID derivado es determinístico (mismo keypair → mismo PeerID)

- [ ] Definir interfaz `SecretsClient` (para poder mockear en tests sin AWS real):

  ```go
  type SecretsClient interface {
      GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, ...) (*secretsmanager.GetSecretValueOutput, error)
      CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, ...) (*secretsmanager.CreateSecretOutput, error)
  }
  ```

---

### Paso 1.2 — Construcción del Host go-libp2p

- [ ] Implementar `internal/node/host.go`:

  ```go
  type HostConfig struct {
      Identity       *NodeIdentity
      ListenTCP      string        // /ip4/0.0.0.0/tcp/4001
      ListenQUIC     string        // /ip4/0.0.0.0/udp/4001/quic-v1
      ConnMgrLow     int           // 900
      ConnMgrHigh    int           // 1000
      ConnMgrGrace   time.Duration // 30s
  }

  // BuildHost construye y retorna el host go-libp2p configurado.
  // No inicia behaviours adicionales (DHT, relay, etc.) — esos se agregan en fases siguientes.
  func BuildHost(ctx context.Context, cfg HostConfig, logger zerolog.Logger) (host.Host, error)
  ```

  Configuración del host:

  ```go
  host, err := libp2p.New(
      libp2p.Identity(cfg.Identity.PrivKey),
      libp2p.ListenAddrStrings(cfg.ListenTCP, cfg.ListenQUIC),
      libp2p.Transport(tcp.NewTCPTransport),
      libp2p.Transport(quic.NewTransport),
      libp2p.Security(noise.ID, noise.New),          // TCP: cifrado Noise
      libp2p.Muxer(yamux.ID, yamux.DefaultTransport), // TCP: multiplexor Yamux
      libp2p.ConnectionManager(connmgr.NewConnManager(
          cfg.ConnMgrLow,
          cfg.ConnMgrHigh,
          connmgr.WithGracePeriod(cfg.ConnMgrGrace),
      )),
      libp2p.NATPortMap(), // intentar UPnP/NAT-PMP automático
  )
  ```

- [ ] Al construir el host, loguear:

  ```json
  {
    "level": "info",
    "event": "host_started",
    "peer_id": "12D3KooW...",
    "listen_addrs": ["/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/udp/4001/quic-v1"],
    "message": "host go-libp2p iniciado"
  }
  ```

- [ ] Integrar `BuildHost` en `cmd/server/main.go`

- [ ] Actualizar `/readyz` para verificar que el host está escuchando:

  ```go
  // ReadinessCheck: el host tiene al menos una dirección de escucha activa
  func HostListeningCheck(h host.Host) ReadinessCheck
  ```

- [ ] Tests de integración para `host.go`:

  ```go
  // TestBuildHost_ArrancarYConectarDosNodos
  // Crea dos hosts en proceso, verifica que pueden hacer ping entre sí
  func TestBuildHost_TwoNodes_CanConnect(t *testing.T) {
      hostA, _ := BuildHost(ctx, cfgA, zerolog.Nop())
      hostB, _ := BuildHost(ctx, cfgB, zerolog.Nop())
      defer hostA.Close()
      defer hostB.Close()

      // Conectar A → B
      err := hostA.Connect(ctx, peer.AddrInfo{
          ID: hostB.ID(), Addrs: hostB.Addrs(),
      })
      require.NoError(t, err)
      assert.Equal(t, network.Connected, hostA.Network().Connectedness(hostB.ID()))
  }
  ```

---

### Paso 1.3 — Protocolo Identify

- [ ] Agregar behaviour `identify` al host en `BuildHost`:

  ```go
  libp2p.UserAgent("bytsers-erp-p2p-cloud/"+version),
  // Identify se añade automáticamente con go-libp2p >= 0.33
  // pero configurarlo explícitamente da control sobre el UserAgent
  ```

- [ ] Loguear cuando un peer completa el handshake Identify:
  - `peer_id`, `user_agent`, `protocol_version`, `listen_addrs`
  - Level: `DEBUG`
- [ ] Test: verificar que dos hosts intercambian Identify correctamente

---

### Paso 1.4 — Protocolo Ping

- [ ] Agregar behaviour `ping` al host:

  ```go
  // Ping se activa implícitamente con go-libp2p pero hay que instanciar el servicio
  ps := ping.NewPingService(host)
  ```

- [ ] Exponer método `PingPeer(ctx, peerID) (time.Duration, error)` en el host wrapper
- [ ] Loguear resultados de ping en DEBUG
- [ ] Test: ping entre dos nodos en proceso, verificar RTT < 10ms (loopback)

---

### Paso 1.5 — Graceful Shutdown del Host

- [ ] Implementar shutdown coordinator en `cmd/server/main.go`:

  ```go
  // Canal de señales OS
  sigCh := make(chan os.Signal, 1)
  signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

  <-sigCh
  log.Info().Msg("señal de shutdown recibida, cerrando...")

  // Shutdown ordenado con timeout
  shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  // 1. Cerrar host go-libp2p (desconecta peers gracefully)
  if err := host.Close(); err != nil {
      log.Error().Err(err).Msg("error al cerrar host")
  }
  // 2. Cerrar servidor HTTP
  httpServer.Shutdown(shutdownCtx)

  log.Info().Msg("shutdown completado")
  ```

- [ ] Test: verificar que el proceso termina limpiamente con SIGTERM en < 5s

---

**✅ Entregables Fase 1:**

- `make dev` arranca el servidor con PeerID impreso en logs
- El host escucha en TCP 4001 y QUIC 4001
- Dos nodos pueden conectarse y hacer ping en tests de integración
- `/readyz` verifica que el host está escuchando activamente
- Identidad Ed25519 cargable desde Secrets Manager y desde archivo local (dev)

---

## Fase 2 — DHT, AutoNAT y Hole Punching (Semanas 3–4)

**Objetivo:** Los nodos ERP-CORE pueden descubrir a otros nodos de su misma organización a través del servidor, y en la mayoría de casos establecer una conexión directa sin relay usando DCUtR.

**Criterio de completitud:** Test de integración con 3 nodos: A registra su PeerID en el DHT, B consulta el DHT y obtiene la dirección de A, ambos intentan hole-punch con éxito en al menos el 60% de las ejecuciones del test.

---

### Paso 2.1 — Kademlia DHT en Modo Servidor

- [ ] Implementar `internal/dht/bootstrap.go`:

  ```go
  type DHTServer struct {
      dht    *dht.IpfsDHT
      host   host.Host
      logger zerolog.Logger
  }

  // New crea e inicia el DHT en modo servidor con el namespace de Bytser.
  func New(ctx context.Context, h host.Host, namespace string, logger zerolog.Logger) (*DHTServer, error)
  ```

  Configuración del DHT:

  ```go
  d, err := dht.New(ctx, h,
      dht.Mode(dht.ModeServer),            // siempre en modo servidor (nunca cliente)
      dht.ProtocolPrefix(protocol.ID(namespace)), // /bytser/erp
      dht.BootstrapPeers(),                // el relay no tiene bootstrap peers propios
      dht.BucketSize(20),                  // K-bucket estándar de Kademlia
  )
  if err != nil { return nil, err }

  // Iniciar proceso de bootstrap del DHT
  if err := d.Bootstrap(ctx); err != nil { return nil, err }
  ```

- [ ] Loguear al iniciar:

  ```json
  {
    "event": "dht_started",
    "mode": "server",
    "namespace": "/bytser/erp",
    "message": "Kademlia DHT iniciado en modo servidor"
  }
  ```

- [ ] Exponer métodos de diagnóstico:

  ```go
  func (s *DHTServer) RoutingTableSize() int
  func (s *DHTServer) PeersInTable() []peer.ID
  ```

- [ ] Actualizar `/readyz` con check del DHT:

  ```go
  func DHTBootstrappedCheck(d *DHTServer) ReadinessCheck
  // Retorna ready=true si el DHT ha completado al menos un ciclo de bootstrap
  ```

- [ ] Tests de integración `dht_test.go`:

  ```go
  // TestDHT_DosNodosSePuedEncontrar
  // Nodo A arranca como servidor DHT
  // Nodo B se conecta al servidor y publica su PeerID en el DHT
  // Nodo C consulta el DHT y encuentra a Nodo B
  func TestDHT_Integration_TwoClientsFindEachOther(t *testing.T)
  ```

---

### Paso 2.2 — AutoNAT Service

- [ ] Implementar `internal/autonat/service.go`:

  ```go
  type AutoNATService struct {
      service autonat.AutoNAT
      logger  zerolog.Logger
  }

  // New crea el servicio AutoNAT en modo servidor (responde sondeos de peers).
  func New(ctx context.Context, h host.Host, throttlePeer time.Duration, logger zerolog.Logger) (*AutoNATService, error)
  ```

  Configuración:

  ```go
  // En go-libp2p, el servidor AutoNAT se habilita cuando el host tiene
  // una IP pública reachable. Se configura con opciones de throttling:
  autonat.EnableService(h,
      autonat.WithSchedule(throttlePeer, throttlePeer),
  )
  ```

- [ ] Loguear cada sondeo AutoNAT recibido en DEBUG:

  ```json
  {
    "event": "autonat_probe",
    "requester_peer": "12D3KooW...",
    "result": "reachable|not_reachable",
    "requester_observed_addr": "/ip4/X.X.X.X/tcp/Y"
  }
  ```

- [ ] Tests:
  - Test: un nodo detrás de NAT simulado recibe respuesta de AutoNAT con su IP pública
  - Test: throttling funciona (segundo sondeo del mismo peer en < 30s retorna error)

---

### Paso 2.3 — DCUtR (Hole Punching)

- [ ] Habilitar DCUtR en el host en `BuildHost`:

  ```go
  libp2p.EnableHolePunching(),
  // DCUtR requiere que el host tenga al menos un relay address reservado
  // Los nodos clientes lo configuran en ERP-CORE; el servidor solo coordina
  ```

- [ ] El servidor actúa como punto de coordinación del hole-punch (no como participante activo).
  go-libp2p maneja esto automáticamente cuando el host tiene el protocolo DCUtR habilitado.

- [ ] Loguear eventos DCUtR en INFO:

  ```json
  {
    "event": "hole_punch_coordinated",
    "peer_a": "12D3KooW...",
    "peer_b": "12D3KooW...",
    "success": true,
    "duration_ms": 450
  }
  ```

- [ ] Implementar tracking de métricas en `internal/metrics/collector.go`:

  ```go
  p2p_hole_punch_attempts_total   // contador
  p2p_hole_punch_success_total    // contador
  // Tasa de éxito = success / attempts (calculable en Grafana)
  ```

- [ ] Tests de integración DCUtR:

  ```go
  // TestDCUtR_DosPeersDetrasDeNATSimulado_ConexionDirecta
  // Usa go-libp2p test helpers para simular NAT
  // Verifica que hole-punch es intentado
  // Verifica que en loopback siempre tiene éxito
  func TestDCUtR_Integration_HolePunchLoopback(t *testing.T)
  ```

---

### Paso 2.4 — Integración Completa del Nodo (node.go)

- [ ] Implementar `internal/node/host.go` como orquestador completo:

  ```go
  // Node es el servidor P2P completo con todos sus behaviours
  type Node struct {
      Host    host.Host
      DHT     *dht.DHTServer
      AutoNAT *autonat.AutoNATService
      // Relay y métricas se agregan en fases siguientes
      logger  zerolog.Logger
  }

  // New construye el nodo completo: host + DHT + AutoNAT + DCUtR
  func New(ctx context.Context, cfg *config.Config, logger zerolog.Logger) (*Node, error)

  // Start inicia todos los servicios y el event loop
  func (n *Node) Start(ctx context.Context) error

  // Stop para todos los servicios limpiamente
  func (n *Node) Stop() error

  // PeerID retorna el identificador del nodo
  func (n *Node) PeerID() peer.ID

  // ListenAddrs retorna las direcciones de escucha actuales
  func (n *Node) ListenAddrs() []multiaddr.Multiaddr

  // ReadinessChecks retorna los checks para /readyz
  func (n *Node) ReadinessChecks() []health.ReadinessCheck
  ```

- [ ] Integrar `Node` en `cmd/server/main.go` (reemplazar el host simple del Paso 1.2)
- [ ] Tests de integración del nodo completo:
  - 3 nodos: A (servidor), B y C (clientes) — B y C se descubren via DHT del servidor A
  - B puede conectarse a C tras descubrir su PeerID

---

### Paso 2.5 — Logging del Event Loop

- [ ] Implementar listener de eventos de conexión del host:

  ```go
  // Suscribirse a eventos del network de go-libp2p
  sub, _ := h.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
  go func() {
      for evt := range sub.Out() {
          e := evt.(event.EvtPeerConnectednessChanged)
          if e.Connectedness == network.Connected {
              log.Info().
                  Str("peer_id", e.Peer.String()).
                  Str("event", "peer_connected").
                  Msg("nuevo peer conectado")
          } else {
              log.Info().
                  Str("peer_id", e.Peer.String()).
                  Str("event", "peer_disconnected").
                  Msg("peer desconectado")
          }
      }
  }()
  ```

---

**✅ Entregables Fase 2:**

- DHT Kademlia funcionando — nodos pueden publicar y descubrir PeerIDs
- AutoNAT funcionando — el servidor informa IP pública a los peers
- DCUtR habilitado — hole punching coordinado por el servidor
- Tests de integración con 3 nodos pasando
- Métricas de hole-punch registradas

---

## Fase 3 — Circuit Relay v2 (Semana 5)

**Objetivo:** Los nodos que no pueden hacer hole-punch (NAT simétrico, firewalls corporativos) pueden establecer una conexión a través del servidor relay. El relay tiene límites estrictos para proteger contra abuso y controlar costos de AWS.

**Criterio de completitud:** Test donde dos nodos con hole-punch artificialmente bloqueado se conectan via relay, con el relay registrando bytes transferidos en las métricas.

---

### Paso 3.1 — Circuit Relay v2 Service

- [ ] Implementar `internal/relay/service.go`:

  ```go
  type RelayService struct {
      relay  *relayv2.Relay
      logger zerolog.Logger
      metrics *RelayMetrics
  }

  type RelayLimits struct {
      MaxReservations     int
      MaxCircuits         int
      ReservationTTL      time.Duration
      MaxCircuitDuration  time.Duration
      MaxCircuitBytes     int64         // 128KB = 131072
  }

  // New crea e inicia el servicio Circuit Relay v2 con los límites configurados.
  func New(h host.Host, limits RelayLimits, logger zerolog.Logger) (*RelayService, error)
  ```

  Configuración del relay:

  ```go
  r, err := relayv2.New(h,
      relayv2.WithLimit(&relayv2.RelayLimit{
          Duration: limits.MaxCircuitDuration,
          Data:     uint64(limits.MaxCircuitBytes),
      }),
      relayv2.WithResources(relayv2.Resources{
          MaxReservations:        limits.MaxReservations,
          MaxCircuits:            limits.MaxCircuits,
          MaxReservationsPerPeer: 4,       // max 4 slots por mismo peer
          MaxReservationsPerIP:   8,       // max 8 slots por misma IP
          ReservationTTL:         limits.ReservationTTL,
          BufferSize:             2048,
      }),
  )
  ```

- [ ] Loguear eventos del relay:

  ```json
  // Reserva de slot
  { "event": "relay_slot_reserved", "peer_id": "...", "ttl": "1h0m0s" }

  // Apertura de circuito
  { "event": "relay_circuit_opened", "src": "...", "dst": "...", "circuit_id": "..." }

  // Cierre de circuito
  { "event": "relay_circuit_closed", "circuit_id": "...", "bytes_relayed": 4096, "duration_s": 12 }

  // Slot rechazado (límites alcanzados)
  { "level": "warn", "event": "relay_slot_rejected", "peer_id": "...", "reason": "max_reservations_reached" }
  ```

- [ ] Actualizar `/readyz` con check del relay:

  ```go
  func RelayActiveCheck(r *RelayService) ReadinessCheck
  ```

---

### Paso 3.2 — Métricas del Relay

- [ ] Implementar `internal/relay/metrics.go` con los contadores Prometheus del relay:

  ```go
  type RelayMetrics struct {
      reservationsActive  prometheus.Gauge
      circuitsActive      prometheus.Gauge
      bytesRelayedTotal   prometheus.Counter
      circuitsOpenedTotal prometheus.Counter
      circuitsClosedTotal prometheus.Counter
      slotsRejectedTotal  prometheus.Counter
  }
  ```

- [ ] Los circuitos activos deben actualizarse en tiempo real (incrementar al abrir, decrementar al cerrar)
- [ ] `bytesRelayedTotal` se actualiza al cerrar cada circuito con los bytes reales transferidos
- [ ] Test: abrir 5 circuitos, verificar que el gauge `relay_circuits_active` == 5

---

### Paso 3.3 — Tests de Circuit Relay

- [ ] Test de integración principal:

  ```go
  // TestCircuitRelay_DosPeersNoPuedenConectarseDirectamente_UsanRelay
  func TestCircuitRelay_Integration_FallbackWhenDirectFails(t *testing.T) {
      // 1. Crear servidor relay (este servidor)
      relayServer := createTestNode(t, withRelayService())

      // 2. Crear Peer A que solo conoce al relay (sin IP pública)
      peerA := createTestNode(t, withRelay(relayServer))

      // 3. Crear Peer B que solo conoce al relay (sin IP pública)
      peerB := createTestNode(t, withRelay(relayServer))

      // 4. Peer A reserva slot en el relay
      relayAddr := relayAddrFor(relayServer, peerB)

      // 5. Peer A conecta a Peer B VÍA relay
      err := peerA.Connect(ctx, peer.AddrInfo{
          ID: peerB.ID(), Addrs: []multiaddr.Multiaddr{relayAddr},
      })
      require.NoError(t, err)

      // 6. Verificar métricas
      assert.Equal(t, 1, getCircuitsActive(relayServer))
      assert.Equal(t, float64(1), getMetric("p2p_relay_circuits_active"))
  }
  ```

- [ ] Test de límites:

  ```go
  // TestCircuitRelay_MaxCircuitsAlcanzado_RechazaNuevos
  func TestCircuitRelay_Integration_RejectsWhenMaxReached(t *testing.T)
  ```

- [ ] Test de expiración de TTL:

  ```go
  // TestCircuitRelay_SlotExpira_SeLibera
  func TestCircuitRelay_Integration_SlotExpires(t *testing.T)
  ```

---

### Paso 3.4 — Integrar Relay en el Nodo Principal

- [ ] Agregar `RelayService` al struct `Node` en `internal/node/host.go`
- [ ] Actualizar `Node.New()` para inicializar el relay con los límites de config
- [ ] Actualizar `Node.Stop()` para cerrar el relay limpiamente
- [ ] Actualizar `Node.ReadinessChecks()` para incluir el check del relay

---

**✅ Entregables Fase 3:**

- Circuit Relay v2 funcionando con todos los límites configurados
- Tests de integración: relay como fallback cuando hole-punch falla
- Métricas de bytes relayados registradas correctamente
- El nodo completo (DHT + AutoNAT + DCUtR + Relay) funciona en tests E2E con 3 peers

---

## Fase 4 — Observabilidad Completa (Semana 6)

**Objetivo:** El servidor es completamente observable en producción. Cualquier anomalía es detectable antes de que los clientes lo reporten. El dashboard de Grafana muestra el estado de la red P2P en tiempo real.

**Criterio de completitud:** Dashboard de Grafana corriendo localmente con docker-compose mostrando todas las métricas del servidor corriendo en dev, con datos realistas de un test de carga básico.

---

### Paso 4.1 — Métricas Prometheus Completas

- [ ] Implementar `internal/metrics/collector.go` con todas las métricas definidas en la arquitectura:

  ```go
  type Collector struct {
      // Conexiones
      ActiveWebSocketConns  prometheus.Gauge    // p2p_active_websocket_connections
      PeersConnectedTotal   prometheus.Counter  // p2p_peers_connected_total (acumulado)
      PeersCurrentlyConn    prometheus.Gauge    // p2p_peers_currently_connected

      // DHT
      DHTRoutingTableSize   prometheus.Gauge    // p2p_dht_peers_in_routing_table

      // Hole Punching
      HolePunchAttempts     prometheus.Counter  // p2p_hole_punch_attempts_total
      HolePunchSuccess      prometheus.Counter  // p2p_hole_punch_success_total

      // AutoNAT
      AutoNATReachable      prometheus.Counter  // p2p_autonat_reachable_total
      AutoNATBehindNAT      prometheus.Counter  // p2p_autonat_nat_total

      // Relay (delegadas al RelayMetrics)
      // p2p_relay_reservations_active
      // p2p_relay_circuits_active
      // p2p_turn_bytes_relayed_total
      // p2p_relay_circuits_opened_total

      // Process
      UptimeSeconds         prometheus.Gauge    // p2p_uptime_seconds
      StartTime             time.Time
  }

  // StartCollection inicia goroutine que actualiza gauges cada 15s
  func (c *Collector) StartCollection(ctx context.Context, node *node.Node)
  ```

- [ ] Implementar `internal/metrics/handler.go`:

  ```go
  // Retorna handler HTTP para /metrics (usa promhttp.Handler())
  func NewPrometheusHandler() http.Handler
  ```

- [ ] Registrar todas las métricas con el registry de Prometheus al inicializar el Collector
- [ ] Tests: verificar que cada métrica es actualizada correctamente por los eventos del nodo

---

### Paso 4.2 — Servidor HTTP de Métricas y Health

- [ ] Implementar servidor HTTP dedicado en `cmd/server/main.go`:

  ```go
  // Puerto 8080: health endpoints (público vía ALB interno)
  healthMux := http.NewServeMux()
  healthMux.HandleFunc("/healthz", health.HandleHealthz)
  healthMux.Handle("/readyz", health.HandleReadyz(node.ReadinessChecks()...))

  // Puerto 9090: métricas Prometheus (solo red interna, no en ALB)
  metricsMux := http.NewServeMux()
  metricsMux.Handle("/metrics", metrics.NewPrometheusHandler())

  // Ambos servidores en goroutines separadas
  go http.ListenAndServe(":8080", healthMux)
  go http.ListenAndServe(":9090", metricsMux)
  ```

- [ ] Test E2E: hacer `curl localhost:9090/metrics` y verificar que las métricas están presentes en formato Prometheus correcto

---

### Paso 4.3 — Logging Estructurado Completo (zerolog)

- [ ] Implementar `internal/config/logging.go`:

  ```go
  // InitLogger configura el logger global según el entorno
  func InitLogger(cfg config.LoggingConfig) zerolog.Logger {
      // Producción: JSON puro para CloudWatch
      // Dev: pretty print con colores
      // Siempre incluir: service, version, peer_id (campos fijos)
  }
  ```

- [ ] Crear helper `logEvent` que añade campos estándar a todos los logs:

  ```go
  func logEvent(logger zerolog.Logger) *zerolog.Event {
      return logger.With().
          Str("service", "bytsers-erp-p2p-cloud").
          Str("version", version).
          Logger().Info()
  }
  ```

- [ ] Auditar todos los módulos para verificar que usan los campos estándar definidos en la arquitectura: `service`, `version`, `peer_id`, `event`, `duration_ms`

- [ ] Test: verificar que el output en modo `json` es JSON válido parseable

---

### Paso 4.4 — Docker Compose de Desarrollo con Grafana

- [ ] Crear `deployments/docker/docker-compose.yml` con:

  ```yaml
  services:
    p2p-server:
      build: { context: ../../, dockerfile: deployments/docker/Dockerfile }
      ports: ["4001:4001/tcp", "4001:4001/udp", "8080:8080", "9090:9090"]
      environment: [APP_ENV=dev, LOG_LEVEL=debug]

    prometheus:
      image: prom/prometheus:latest
      ports: ["9091:9090"]
      volumes:
        - ./prometheus.yml:/etc/prometheus/prometheus.yml
      # prometheus.yml apunta a http://p2p-server:9090/metrics

    grafana:
      image: grafana/grafana:latest
      ports: ["3000:3000"]
      volumes:
        - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
        - ./grafana/datasources:/etc/grafana/provisioning/datasources
  ```

- [ ] Crear dashboard de Grafana como JSON en `deployments/grafana/dashboard_p2p.json` con los paneles:
  - **Peers conectados actualmente** (gauge)
  - **Bytes triangulados (TURN)** (time series) — métrica de costo AWS
  - **Circuitos relay activos** (gauge con alerta si > 55)
  - **Tasa de éxito hole-punch** (stat: success/attempts * 100)
  - **Peers en tabla DHT** (gauge)
  - **Conexiones WebSocket activas** (gauge)
  - **Uptime del servidor** (stat)

- [ ] Verificar: `make dev` → `docker-compose up` → abrir `localhost:3000` → dashboard visible con datos

---

### Paso 4.5 — Dockerfile de Producción

- [ ] Crear `deployments/docker/Dockerfile` multi-stage optimizado:

  ```dockerfile
  # ─── Stage 1: Build ───────────────────────────────────────────
  FROM golang:1.22-alpine AS builder

  # Instalar dependencias de compilación (go-libp2p puede necesitar gcc para QUIC)
  RUN apk add --no-cache gcc musl-dev

  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download

  COPY . .

  ARG VERSION=dev
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
      go build \
      -ldflags="-w -s -X main.version=${VERSION}" \
      -trimpath \
      -o /erp-p2p-cloud \
      ./cmd/server

  # ─── Stage 2: Imagen final (~15MB) ────────────────────────────
  FROM alpine:3.19

  # ca-certificates: necesario para TLS con Secrets Manager y QUIC
  # tzdata: para logs con timezone correcta
  RUN apk --no-cache add ca-certificates tzdata

  WORKDIR /app
  COPY --from=builder /erp-p2p-cloud .

  # Puerto TCP y UDP para libp2p
  EXPOSE 4001/tcp
  EXPOSE 4001/udp
  # Puerto health
  EXPOSE 8080
  # Puerto métricas
  EXPOSE 9090

  # Usuario no-root por seguridad
  RUN addgroup -S appgroup && adduser -S appuser -G appgroup
  USER appuser

  ENTRYPOINT ["/app/erp-p2p-cloud"]
  ```

- [ ] Verificar: imagen final < 20MB, `docker run --rm image /app/erp-p2p-cloud --help` funciona

---

**✅ Entregables Fase 4:**

- Todas las métricas Prometheus implementadas y accesibles en `/metrics`
- Dashboard Grafana con todos los paneles operativos
- Docker Compose local con Prometheus + Grafana funcionando
- Dockerfile de producción multi-stage < 20MB
- Logs estructurados JSON en todos los módulos

---

## Fase 5 — Infraestructura AWS y IaC (Semana 7)

**Objetivo:** La infraestructura en AWS está completamente descrita como código (Terraform) y el servidor corre en una instancia EC2 real en staging con el dominio `relay.bytsers.com` apuntando a él.

**Criterio de completitud:** `relay.bytsers.com:4001` responde a conexiones go-libp2p desde una máquina local, con el wildcard ACM `*.bytsers.com` cubriendo el endpoint HTTPS de health.

---

### Paso 5.1 — Backend de Estado de Terraform

- [ ] Crear manualmente en AWS (una única vez, fuera de Terraform):
  - S3 bucket `bytsers-terraform-state` con versionado habilitado y acceso público bloqueado
  - DynamoDB table `bytsers-terraform-lock` con `LockID` como partition key (tipo String)
- [ ] Crear `deployments/terraform/main.tf`:

  ```hcl
  terraform {
    required_version = ">= 1.7"
    required_providers {
      aws = {
        source  = "hashicorp/aws"
        version = "~> 5.0"
      }
    }
    backend "s3" {
      bucket         = "bytsers-terraform-state"
      key            = "erp-p2p-cloud/terraform.tfstate"
      region         = "us-east-1"
      encrypt        = true
      dynamodb_table = "bytsers-terraform-lock"
    }
  }

  provider "aws" {
    region = var.aws_region
    default_tags {
      tags = {
        Project     = "bytsers-erp-p2p-cloud"
        ManagedBy   = "terraform"
        Environment = var.environment
      }
    }
  }
  ```

---

### Paso 5.2 — Variables y Outputs

- [ ] Crear `deployments/terraform/variables.tf`:

  ```hcl
  variable "aws_region"      { default = "us-east-1" }
  variable "environment"     { default = "staging" }
  variable "instance_type"   { default = "t4g.nano" }
  variable "ssh_key_name"    { description = "Nombre del key pair EC2 para SSH" }
  variable "allowed_ssh_cidr" { description = "CIDR permitido para SSH (tu IP)" }
  variable "domain_name"     { default = "bytsers.com" }
  variable "subdomain"       { default = "relay" }
  variable "ecr_image_tag"   { default = "latest" }
  variable "acm_cert_arn"    {
    description = "ARN del wildcard cert *.bytsers.com en ACM (ya provisionado)"
  }
  ```

- [ ] Crear `deployments/terraform/outputs.tf`:

  ```hcl
  output "instance_id"        { value = aws_instance.relay.id }
  output "elastic_ip"         { value = aws_eip.relay.public_ip }
  output "relay_domain"       { value = "${var.subdomain}.${var.domain_name}" }
  output "relay_multiaddr_tcp" {
    value = "/dns4/${var.subdomain}.${var.domain_name}/tcp/4001/p2p/${var.relay_peer_id}"
  }
  output "relay_multiaddr_quic" {
    value = "/dns4/${var.subdomain}.${var.domain_name}/udp/4001/quic-v1/p2p/${var.relay_peer_id}"
  }
  output "ssh_command"        {
    value = "ssh ubuntu@${aws_eip.relay.public_ip}"
  }
  ```

---

### Paso 5.3 — VPC y Red

- [ ] Crear `deployments/terraform/vpc.tf`:

  ```hcl
  # VPC mínima — una sola subnet pública
  # El servidor P2P necesita IP pública directa para que los peers puedan conectarse

  resource "aws_vpc" "main" {
    cidr_block           = "10.0.0.0/24"
    enable_dns_hostnames = true
    enable_dns_support   = true
  }

  resource "aws_subnet" "public" {
    vpc_id                  = aws_vpc.main.id
    cidr_block              = "10.0.0.0/24"
    availability_zone       = "${var.aws_region}a"
    map_public_ip_on_launch = false  # usamos EIP fija
  }

  resource "aws_internet_gateway" "igw" {
    vpc_id = aws_vpc.main.id
  }

  resource "aws_route_table" "public" {
    vpc_id = aws_vpc.main.id
    route {
      cidr_block = "0.0.0.0/0"
      gateway_id = aws_internet_gateway.igw.id
    }
  }

  resource "aws_route_table_association" "public" {
    subnet_id      = aws_subnet.public.id
    route_table_id = aws_route_table.public.id
  }

  # IP Elástica fija — CRÍTICA para PeerID estable
  resource "aws_eip" "relay" {
    domain = "vpc"
    tags   = { Name = "bytsers-p2p-relay-eip" }
  }

  resource "aws_eip_association" "relay" {
    instance_id   = aws_instance.relay.id
    allocation_id = aws_eip.relay.id
  }
  ```

---

### Paso 5.4 — Security Groups

- [ ] Crear `deployments/terraform/security_groups.tf`:

  ```hcl
  resource "aws_security_group" "relay" {
    name        = "bytsers-p2p-relay-sg"
    description = "Security group para el servidor P2P relay"
    vpc_id      = aws_vpc.main.id

    # P2P TCP — abierto a internet para conexiones de peers
    ingress {
      description = "libp2p TCP"
      from_port   = 4001
      to_port     = 4001
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }

    # P2P QUIC — abierto a internet para conexiones de peers
    ingress {
      description = "libp2p QUIC"
      from_port   = 4001
      to_port     = 4001
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

    # SSH — solo desde IP autorizada
    ingress {
      description = "SSH admin"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = [var.allowed_ssh_cidr]
    }

    # Health y métricas — solo dentro de la VPC
    ingress {
      description = "Health check interno"
      from_port   = 8080
      to_port     = 8080
      protocol    = "tcp"
      cidr_blocks = [aws_vpc.main.cidr_block]
    }

    ingress {
      description = "Métricas Prometheus (interno)"
      from_port   = 9090
      to_port     = 9090
      protocol    = "tcp"
      cidr_blocks = [aws_vpc.main.cidr_block]
    }

    # Todo el tráfico saliente permitido
    egress {
      from_port   = 0
      to_port     = 0
      protocol    = "-1"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
  ```

---

### Paso 5.5 — IAM Role de EC2

- [ ] Crear `deployments/terraform/iam.tf`:

  ```hcl
  # Rol con permisos mínimos: solo Secrets Manager + CloudWatch Logs
  resource "aws_iam_role" "relay" {
    name = "bytsers-p2p-relay-role"
    assume_role_policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Action    = "sts:AssumeRole"
        Effect    = "Allow"
        Principal = { Service = "ec2.amazonaws.com" }
      }]
    })
  }

  resource "aws_iam_role_policy" "relay" {
    name = "bytsers-p2p-relay-policy"
    role = aws_iam_role.relay.id
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          # Solo leer el secreto específico del keypair del relay
          Effect   = "Allow"
          Action   = ["secretsmanager:GetSecretValue"]
          Resource = "arn:aws:secretsmanager:${var.aws_region}:*:secret:bytsers/p2p-relay/identity*"
        },
        {
          # Escribir logs a CloudWatch
          Effect = "Allow"
          Action = [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents",
            "logs:DescribeLogStreams"
          ]
          Resource = "arn:aws:logs:${var.aws_region}:*:log-group:/bytsers/p2p-cloud:*"
        }
      ]
    })
  }

  resource "aws_iam_instance_profile" "relay" {
    name = "bytsers-p2p-relay-profile"
    role = aws_iam_role.relay.name
  }
  ```

---

### Paso 5.6 — Instancia EC2 y User Data

- [ ] Crear `deployments/terraform/ec2.tf`:

  ```hcl
  # AMI Ubuntu 24.04 ARM64 (Graviton2)
  data "aws_ami" "ubuntu_arm" {
    most_recent = true
    owners      = ["099720109477"] # Canonical
    filter {
      name   = "name"
      values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"]
    }
    filter {
      name   = "architecture"
      values = ["arm64"]
    }
  }

  resource "aws_instance" "relay" {
    ami                    = data.aws_ami.ubuntu_arm.id
    instance_type          = var.instance_type  # t4g.nano
    key_name               = var.ssh_key_name
    subnet_id              = aws_subnet.public.id
    vpc_security_group_ids = [aws_security_group.relay.id]
    iam_instance_profile   = aws_iam_instance_profile.relay.name

    root_block_device {
      volume_type           = "gp3"
      volume_size           = 8  # GB — solo el SO + Docker + binario (~2GB usado)
      delete_on_termination = true
      encrypted             = true
    }

    user_data = base64encode(templatefile("${path.module}/user_data.sh", {
      ecr_repo      = var.ecr_repo
      image_tag     = var.ecr_image_tag
      aws_region    = var.aws_region
      app_env       = var.environment
    }))

    tags = { Name = "bytsers-p2p-relay-${var.environment}" }
  }
  ```

- [ ] Crear `deployments/terraform/user_data.sh`:

  ```bash
  #!/bin/bash
  set -euo pipefail

  # Instalar Docker
  apt-get update -q
  apt-get install -y docker.io awscli
  systemctl enable --now docker

  # Instalar CloudWatch agent
  wget -q https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/arm64/latest/amazon-cloudwatch-agent.deb
  dpkg -i amazon-cloudwatch-agent.deb

  # Login a ECR y pull de la imagen
  aws ecr get-login-password --region ${aws_region} | \
      docker login --username AWS --password-stdin ${ecr_repo}
  docker pull ${ecr_repo}:${image_tag}

  # Crear archivo de configuración del servidor
  cat > /opt/erp-p2p-cloud.env << EOF
  APP_ENV=${app_env}
  LOG_LEVEL=info
  AWS_DEFAULT_REGION=${aws_region}
  EOF

  # Crear systemd service para auto-restart
  cat > /etc/systemd/system/erp-p2p-cloud.service << EOF
  [Unit]
  Description=Bytsers ERP P2P Cloud Relay
  After=docker.service
  Requires=docker.service

  [Service]
  Restart=always
  RestartSec=10
  ExecStartPre=-/usr/bin/docker stop erp-p2p-cloud
  ExecStartPre=-/usr/bin/docker rm erp-p2p-cloud
  ExecStart=/usr/bin/docker run \
      --name erp-p2p-cloud \
      --env-file /opt/erp-p2p-cloud.env \
      -p 4001:4001/tcp \
      -p 4001:4001/udp \
      -p 8080:8080 \
      -p 9090:9090 \
      --log-driver awslogs \
      --log-opt awslogs-region=${aws_region} \
      --log-opt awslogs-group=/bytsers/p2p-cloud \
      --log-opt awslogs-create-group=true \
      ${ecr_repo}:${image_tag}
  ExecStop=/usr/bin/docker stop erp-p2p-cloud

  [Install]
  WantedBy=multi-user.target
  EOF

  systemctl enable --now erp-p2p-cloud
  ```

---

### Paso 5.7 — Route53 y CloudWatch

- [ ] Crear `deployments/terraform/route53.tf`:

  ```hcl
  # Usar la hosted zone del dominio bytsers.com (ya existe)
  data "aws_route53_zone" "main" {
    name         = "bytsers.com."
    private_zone = false
  }

  # Registro A: relay.bytsers.com → EIP del servidor
  resource "aws_route53_record" "relay" {
    zone_id = data.aws_route53_zone.main.zone_id
    name    = "relay.${data.aws_route53_zone.main.name}"
    type    = "A"
    ttl     = 300
    records = [aws_eip.relay.public_ip]
  }
  ```

  > **Nota ACM:** El wildcard certificate `*.bytsers.com` ya está provisionado en ACM. El subdominio `relay.bytsers.com` queda cubierto automáticamente. Este certificado se referencia en el output como `var.acm_cert_arn` para futuros ALBs si se necesitan.

- [ ] Crear `deployments/terraform/cloudwatch.tf`:

  ```hcl
  resource "aws_cloudwatch_log_group" "p2p_cloud" {
    name              = "/bytsers/p2p-cloud"
    retention_in_days = 30
  }

  # Alarma: CPU alto → posible abuso del relay
  resource "aws_cloudwatch_metric_alarm" "high_cpu" {
    alarm_name          = "bytsers-p2p-relay-high-cpu"
    comparison_operator = "GreaterThanThreshold"
    evaluation_periods  = 2
    metric_name         = "CPUUtilization"
    namespace           = "AWS/EC2"
    period              = 300
    statistic           = "Average"
    threshold           = 80
    alarm_description   = "CPU > 80% en el servidor P2P relay por 10 minutos"
    dimensions          = { InstanceId = aws_instance.relay.id }
    alarm_actions       = [var.sns_alert_topic_arn]
    ok_actions          = [var.sns_alert_topic_arn]
  }
  ```

---

### Paso 5.8 — ECR Repository

- [ ] Crear `deployments/terraform/ecr.tf`:

  ```hcl
  resource "aws_ecr_repository" "relay" {
    name                 = "erp-p2p-cloud"
    image_tag_mutability = "MUTABLE"
    force_delete         = false

    image_scanning_configuration {
      scan_on_push = true
    }

    encryption_configuration {
      encryption_type = "AES256"
    }
  }

  # Política de lifecycle: retener solo los últimos 10 tags
  resource "aws_ecr_lifecycle_policy" "relay" {
    repository = aws_ecr_repository.relay.name
    policy = jsonencode({
      rules = [{
        rulePriority = 1
        description  = "Retener últimas 10 imágenes"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = 10
        }
        action = { type = "expire" }
      }]
    })
  }
  ```

---

### Paso 5.9 — Primer Deploy en Staging

- [ ] Ejecutar: `make tf-plan` — revisar que los recursos son los esperados
- [ ] Ejecutar: `make tf-apply` — aprovisionar toda la infraestructura
- [ ] Ejecutar: `make gen-identity` — generar keypair y guardarlo en Secrets Manager
- [ ] Anotar el PeerID generado en el README y en la configuración de bootstrap de ERP-CORE
- [ ] Verificar desde una máquina local:
  - `curl http://relay.bytsers.com:8080/healthz` → `{"status": "ok"}`
  - `curl http://relay.bytsers.com:8080/readyz` → `{"status": "ready", "checks": {...}}`
  - Conectar un nodo go-libp2p de test a `relay.bytsers.com:4001`

---

**✅ Entregables Fase 5:**

- Infraestructura completa en AWS gestionada por Terraform
- `relay.bytsers.com` apuntando a la instancia EC2 en staging
- `/healthz` y `/readyz` respondiendo desde internet
- Nodo go-libp2p de prueba puede conectarse al relay en staging
- CloudWatch Logs recibiendo logs estructurados del servidor

---

## Fase 6 — CI/CD y Pipeline de Despliegue (Semana 8)

**Objetivo:** El proceso de despliegue es completamente automatizado. Un merge a `main` despliega automáticamente a staging. Un despliegue a producción se hace con un click desde GitHub Actions con aprobación manual.

**Criterio de completitud:** Hacer un cambio trivial en el código, mergear a main, y verificar que llega automáticamente a staging en < 5 minutos sin intervención manual.

---

### Paso 6.1 — ECR Push en CI

- [ ] Crear `deployments/terraform/github_actions_role.tf`:

  ```hcl
  # OIDC trust para que GitHub Actions asuma un rol AWS sin credenciales estáticas
  resource "aws_iam_openid_connect_provider" "github" {
    url             = "https://token.actions.githubusercontent.com"
    client_id_list  = ["sts.amazonaws.com"]
    thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  }

  resource "aws_iam_role" "github_actions" {
    name = "bytsers-p2p-cloud-github-actions"
    assume_role_policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = "repo:bytsers/erp-p2p-cloud:*"
          }
        }
      }]
    })
  }

  resource "aws_iam_role_policy" "github_actions" {
    name = "bytsers-p2p-cloud-cicd-policy"
    role = aws_iam_role.github_actions.id
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect   = "Allow"
          Action   = ["ecr:GetAuthorizationToken"]
          Resource = "*"
        },
        {
          Effect = "Allow"
          Action = ["ecr:BatchCheckLayerAvailability", "ecr:GetDownloadUrlForLayer",
                    "ecr:BatchGetImage", "ecr:PutImage",
                    "ecr:InitiateLayerUpload", "ecr:UploadLayerPart",
                    "ecr:CompleteLayerUpload"]
          Resource = aws_ecr_repository.relay.arn
        }
      ]
    })
  }
  ```

---

### Paso 6.2 — Script de Deploy

- [ ] Crear `scripts/deploy.sh`:

  ```bash
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

  # SSH al servidor y actualizar el contenedor
  ssh -o StrictHostKeyChecking=no ubuntu@$EC2_IP << EOF
    aws ecr get-login-password --region us-east-1 | \
        docker login --username AWS --password-stdin $ECR_REPO

    # Actualizar el tag en el service file y reiniciar
    sed -i "s|:.*$|:${IMAGE_TAG}|" /etc/systemd/system/erp-p2p-cloud.service
    systemctl daemon-reload
    systemctl restart erp-p2p-cloud

    # Esperar hasta que /readyz responda
    for i in \$(seq 1 12); do
      if curl -sf http://localhost:8080/readyz; then
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
  ```

---

### Paso 6.3 — Workflow: build.yml

- [ ] Crear `.github/workflows/build.yml`:

  ```yaml
  name: Build & Deploy

  on:
    push:
      branches: [main]

  permissions:
    id-token: write   # para OIDC con AWS
    contents: read

  jobs:
    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version: '1.22' }
        - run: go test ./... -race -timeout 120s -coverprofile=coverage.out
        - run: |
            COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
            awk -v c="$COVERAGE" 'BEGIN{if(c+0<80){print "Cobertura "c"% < 80%"; exit 1}}'

    build-push:
      needs: test
      runs-on: ubuntu-latest
      outputs:
        image_tag: ${{ steps.meta.outputs.tag }}
      steps:
        - uses: actions/checkout@v4

        - name: Configurar QEMU (para arm64)
          uses: docker/setup-qemu-action@v3

        - name: Configurar Docker Buildx
          uses: docker/setup-buildx-action@v3

        - name: Configurar credenciales AWS (OIDC)
          uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: arn:aws:iam::ACCOUNT_ID:role/bytsers-p2p-cloud-github-actions
            aws-region: us-east-1

        - name: Login a ECR
          uses: aws-actions/amazon-ecr-login@v2

        - name: Generar tag de imagen
          id: meta
          run: |
            SHORT_SHA=$(git rev-parse --short HEAD)
            echo "tag=${SHORT_SHA}" >> $GITHUB_OUTPUT

        - name: Build y Push (linux/arm64)
          uses: docker/build-push-action@v5
          with:
            context: .
            file: deployments/docker/Dockerfile
            platforms: linux/arm64
            push: true
            tags: |
              ${{ env.ECR_REGISTRY }}/erp-p2p-cloud:${{ steps.meta.outputs.tag }}
              ${{ env.ECR_REGISTRY }}/erp-p2p-cloud:latest
            build-args: VERSION=${{ steps.meta.outputs.tag }}
            cache-from: type=gha
            cache-to: type=gha,mode=max

    deploy-staging:
      needs: build-push
      runs-on: ubuntu-latest
      environment: staging
      steps:
        - uses: actions/checkout@v4

        - name: Configurar credenciales AWS
          uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: arn:aws:iam::ACCOUNT_ID:role/bytsers-p2p-cloud-github-actions
            aws-region: us-east-1

        - name: Agregar SSH key
          run: |
            echo "${{ secrets.STAGING_SSH_KEY }}" > /tmp/key.pem
            chmod 600 /tmp/key.pem

        - name: Deploy a staging
          run: |
            STAGING_IP=$(aws ec2 describe-instances \
              --filters "Name=tag:Name,Values=bytsers-p2p-relay-staging" \
                        "Name=instance-state-name,Values=running" \
              --query 'Reservations[0].Instances[0].PublicIpAddress' \
              --output text)
            ./scripts/deploy.sh staging ${{ needs.build-push.outputs.image_tag }}

        - name: Notificar Slack
          if: always()
          uses: slackapi/slack-github-action@v1
          with:
            payload: |
              {
                "text": "Deploy P2P relay staging: ${{ job.status }} — v${{ needs.build-push.outputs.image_tag }}"
              }
          env:
            SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
  ```

---

### Paso 6.4 — Workflow: deploy.yml (Producción)

- [ ] Crear `.github/workflows/deploy.yml`:

  ```yaml
  name: Deploy a Producción

  on:
    workflow_dispatch:
      inputs:
        image_tag:
          description: 'Tag de imagen ECR a desplegar'
          required: true
        confirm:
          description: 'Escribe "PRODUCCION" para confirmar'
          required: true

  permissions:
    id-token: write
    contents: read
    deployments: write

  jobs:
    validate:
      runs-on: ubuntu-latest
      steps:
        - name: Validar confirmación
          run: |
            if [ "${{ github.event.inputs.confirm }}" != "PRODUCCION" ]; then
              echo "Confirmación incorrecta. Escribe exactamente: PRODUCCION"
              exit 1
            fi

    deploy:
      needs: validate
      runs-on: ubuntu-latest
      environment: production  # requiere aprobación en GitHub Environments
      steps:
        - uses: actions/checkout@v4

        - name: Crear GitHub Deployment
          uses: chrnorm/deployment-action@v2
          id: deployment
          with:
            token: ${{ github.token }}
            environment: production

        - name: Configurar credenciales AWS
          uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: arn:aws:iam::ACCOUNT_ID:role/bytsers-p2p-cloud-github-actions
            aws-region: us-east-1

        - name: Notificar inicio
          uses: slackapi/slack-github-action@v1
          with:
            payload: '{"text": "🚀 Deploy P2P PRODUCCIÓN iniciado: v${{ github.event.inputs.image_tag }} por ${{ github.actor }}"}'
          env:
            SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}

        - name: Agregar SSH key
          run: |
            echo "${{ secrets.PROD_SSH_KEY }}" > /tmp/key.pem
            chmod 600 /tmp/key.pem

        - name: Deploy a producción
          run: ./scripts/deploy.sh prod ${{ github.event.inputs.image_tag }}

        - name: Actualizar estado del deployment
          if: always()
          uses: chrnorm/deployment-status@v2
          with:
            token: ${{ github.token }}
            deployment-id: ${{ steps.deployment.outputs.deployment_id }}
            state: ${{ job.status == 'success' && 'success' || 'failure' }}

        - name: Notificar resultado
          if: always()
          uses: slackapi/slack-github-action@v1
          with:
            payload: '{"text": "${{ job.status == ''success'' && ''✅'' || ''❌'' }} Deploy P2P PRODUCCIÓN: ${{ job.status }} — v${{ github.event.inputs.image_tag }}"}'
          env:
            SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
  ```

---

### Paso 6.5 — GitHub Secrets y Environments

- [ ] Configurar en el repositorio GitHub:
  - **Secrets del repositorio:**
    - `STAGING_SSH_KEY` — clave privada SSH para EC2 staging
    - `PROD_SSH_KEY` — clave privada SSH para EC2 producción
    - `SLACK_WEBHOOK_URL` — webhook para notificaciones
  - **Environment `staging`:** sin aprobación requerida (auto-deploy)
  - **Environment `production`:** requerir aprobación de 1 reviewer + regla: solo desde branch `main`

---

**✅ Entregables Fase 6:**

- Push a `main` → tests → build imagen ARM64 → push ECR → deploy staging: automático en < 5 min
- Deploy a producción: manual con aprobación, un solo click desde GitHub
- Sin credenciales AWS estáticas en el repositorio (OIDC)
- Notificaciones Slack en cada deploy

---

## Fase 7 — Hardening, Tests E2E y Go-Live (Semana 9)

**Objetivo:** El servidor está listo para procesar tráfico P2P de clientes reales. Los tests E2E validan el flujo completo de ICE. La documentación operativa está completa.

**Criterio de completitud:** Al menos un nodo ERP-CORE real puede conectarse a `relay.bytsers.com:4001`, descubrir otro nodo ERP-CORE de la misma organización, e intentar hole-punch.

---

### Paso 7.1 — Tests E2E del Framework ICE Completo

- [ ] Implementar `tests/e2e/ice_ladder_test.go`:

  ```go
  // +build e2e

  // TestICE_LAN_DirectConnection
  // Dos nodos en la misma red (loopback) se descubren por mDNS y conectan directamente
  // El relay no procesa ningún byte
  func TestICE_E2E_LANDirectConnection(t *testing.T) {
      relayNode := createRelayNode(t) // relay local de test
      nodeA := createClientNode(t, relayNode)
      nodeB := createClientNode(t, relayNode)

      // Descubrimiento vía DHT
      publishInDHT(t, nodeA, "test-org-uuid")
      peersFound := queryDHT(t, nodeB, "test-org-uuid")

      require.Contains(t, peersFound, nodeA.ID())
      assert.Equal(t, float64(0), getMetric(relayNode, "p2p_relay_circuits_active"))
  }

  // TestICE_WAN_HolePunchSuccess
  // Dos nodos con NAT simulado hacen hole-punch exitosamente
  // El relay cierra el circuito después del upgrade
  func TestICE_E2E_HolePunchUpgradesFromRelay(t *testing.T)

  // TestICE_WAN_RelayFallback
  // Hole-punch bloqueado → conexión se mantiene via relay
  // relay_circuits_active = 1, bytes_relayed > 0
  func TestICE_E2E_RelayFallbackWhenHolePunchBlocked(t *testing.T)
  ```

---

### Paso 7.2 — Test de Carga Básico

- [ ] Crear `scripts/load_test.go`:
  - Conectar 100 nodos go-libp2p simulados al relay
  - Cada nodo reserva un slot de relay
  - Verificar que los primeros 128 se aceptan y el resto se rechaza
  - Verificar que las métricas Prometheus reflejan los valores correctos
  - Verificar que la memoria del proceso no supera 150MB

---

### Paso 7.3 — Revisión de Seguridad

- [ ] Ejecutar `govulncheck ./...` y resolver todos los CVEs de severidad HIGH/CRITICAL
- [ ] Ejecutar Trivy sobre la imagen Docker: `trivy image bytsers/erp-p2p-cloud:latest`
- [ ] Verificar que el usuario del contenedor es `appuser` (no root): `docker run --rm image id`
- [ ] Verificar que los puertos 9090 (métricas) y 8080 (health) no son accesibles desde internet (solo VPC) usando las reglas de Security Group de Terraform
- [ ] Revisar que no hay credenciales hardcodeadas en ningún archivo del repositorio: `grep -r "aws_secret\|password\|private_key" --include="*.go" --include="*.yaml"`

---

### Paso 7.4 — Documentación Operativa Final

- [ ] Actualizar `README.md` con:
  - Qué hace este servidor y qué NO hace
  - Cómo correr localmente (`make dev`)
  - Cómo ejecutar tests (`make test`)
  - Cómo hacer deploy (`make deploy-staging`, `make deploy-prod`)
  - Multiaddr de bootstrap del relay de producción (con PeerID real)
  - Enlace al dashboard de Grafana

- [ ] Crear `docs/RUNBOOK.md`:
  - Procedimiento de respuesta ante alta CPU
  - Procedimiento de respuesta ante relay saturado
  - Procedimiento de reinicio del servidor
  - Procedimiento de rotación de identidad (con warning de impacto)
  - Cómo interpretar los logs de CloudWatch
  - Cómo acceder al dashboard de Grafana

- [ ] Crear `docs/BOOTSTRAP_CONFIG.md`:
  - El PeerID del relay de producción
  - Las multiaddrs completas para copiar en la config de ERP-CORE:

    ```
    /dns4/relay.bytsers.com/tcp/4001/p2p/12D3KooW[PEER_ID]
    /dns4/relay.bytsers.com/udp/4001/quic-v1/p2p/12D3KooW[PEER_ID]
    ```

---

### Paso 7.5 — Despliegue a Producción

- [ ] Crear la instancia de producción: ejecutar `make tf-apply` con `environment=prod`
- [ ] **No reutilizar** el keypair de staging — ejecutar `make gen-identity` para producción por separado y guardar en un secreto diferente: `bytsers/p2p-relay/identity-prod`
- [ ] Actualizar `docs/BOOTSTRAP_CONFIG.md` con el PeerID de producción real
- [ ] Verificar desde internet:
  - `curl https://relay.bytsers.com/healthz` — usando wildcard ACM (si hay terminación TLS)
  - `curl http://relay.bytsers.com:8080/healthz` — health directo
  - Conexión exitosa de un nodo ERP-CORE real al relay
- [ ] Monitorear las primeras 24 horas activamente con el dashboard de Grafana

---

### Paso 7.6 — Checklist Go-Live Final

- [ ] ✅ `relay.bytsers.com:4001` acepta conexiones TCP y UDP desde internet
- [ ] ✅ `/healthz` retorna 200
- [ ] ✅ `/readyz` retorna 200 con todos los checks pasando
- [ ] ✅ Dashboard Grafana mostrando datos de producción en tiempo real
- [ ] ✅ CloudWatch Logs recibiendo logs con nivel INFO
- [ ] ✅ Alarma de CPU configurada en CloudWatch
- [ ] ✅ Al menos un nodo ERP-CORE real conectado y visible en `p2p_peers_currently_connected`
- [ ] ✅ Cobertura de tests > 80% en el CI
- [ ] ✅ README con multiaddrs de producción publicadas
- [ ] ✅ Runbook disponible para el equipo de soporte
- [ ] ✅ No hay credenciales en el repositorio
- [ ] ✅ El contenedor corre como usuario no-root
- [ ] ✅ Pipeline CI/CD verde de principio a fin

---

**✅ Entregables Fase 7 — Proyecto Completado:**

- Servidor en producción procesando tráfico P2P real
- Tests E2E del framework ICE completo pasando
- Documentación operativa completa
- Pipeline CI/CD funcionando de extremo a extremo
- Dashboard de Grafana operacional con datos reales

---

## Criterios de Éxito Globales

| Métrica | Target | Cómo medirlo |
|---|---|---|
| Tasa de éxito hole-punch | > 60% | `p2p_hole_punch_success_total / p2p_hole_punch_attempts_total` en Grafana |
| Tiempo de descubrimiento DHT | < 10s desde arranque | Logs `dht_peer_found` con timestamp |
| Circuitos relay simultáneos | soportar 64 | Test de carga en Fase 7 |
| Bytes TURN por día (costo AWS) | < 1GB/día inicial | `p2p_turn_bytes_relayed_total` en Grafana |
| Tiempo de recuperación ante falla | < 5 min | Test de kill + medición de reconexión |
| Cobertura de tests | > 80% | `go tool cover` en CI |
| Tiempo de deploy staging | < 5 min | Timestamp en GitHub Actions |
| Memoria del proceso | < 100MB | CloudWatch `MemoryUtilization` |
| CPU en reposo (sin peers) | < 2% | CloudWatch `CPUUtilization` |
| Imagen Docker final | < 20MB | `docker image ls` |

---

## Registro de Decisiones Técnicas Pendientes

Estos puntos deben resolverse durante el desarrollo, no antes:

| # | Decisión | Contexto | Deadline |
|---|---|---|---|
| 1 | Confirmar PeerID de producción | Se genera en Paso 7.5 — debe comunicarse al equipo de ERP-CORE para hardcodear en bootstrap config | Antes de Go-Live |
| 2 | ARN del certificado ACM wildcard | Necesario para `var.acm_cert_arn` en Terraform — obtenerlo de la consola AWS | Fase 5 |
| 3 | CIDR de SSH permitido | `var.allowed_ssh_cidr` — definir IP fija del equipo o usar bastion host | Fase 5 |
| 4 | ARN del SNS topic para alertas | `var.sns_alert_topic_arn` para la alarma de CloudWatch | Fase 5 |
| 5 | Account ID de AWS | Necesario para los ARNs en los roles IAM de GitHub Actions | Fase 6 |
| 6 | Slack Webhook URL | Para notificaciones de deploy en CI/CD | Fase 6 |

---

*Roadmap aprobado junto con la Arquitectura v1.1.*  
*Cualquier cambio de alcance posterior a la aprobación debe actualizar este documento y crear un ADR.*
