# ERP-P2P-CLOUD

Stateless P2P relay server built with [go-libp2p](https://github.com/libp2p/go-libp2p). Enables NAT traversal for ERP desktop nodes using the ICE connection hierarchy: mDNS (LAN) > STUN/hole-punching (DCUtR) > Circuit Relay v2 (fallback).

## Features

- **Ed25519 identity** with AWS Secrets Manager or local file storage
- **Dual DHT** (WAN + LAN) with Kademlia routing and bootstrap peers
- **Circuit Relay v2** with configurable resource limits and abuse prevention
- **AutoNAT** service with forced public reachability
- **Prometheus metrics** (custom `erp_p2p_*` + built-in `libp2p_swarm_*`, `libp2p_relaysvc_*`)
- **Health endpoints** (`/healthz` liveness, `/readyz` readiness)
- **Docker Compose** observability stack (Prometheus + Grafana)
- **CI/CD** with GitHub Actions, OIDC AWS auth, automatic staging deploy

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose (for observability stack)
- golangci-lint v1.64+ (for linting)

### Run locally

```bash
make dev
```

This starts the server with `APP_ENV=dev`, which:
- Generates a local Ed25519 identity in `.local/identity.key`
- Listens on TCP+QUIC port 4001
- Health server on `:8080`, metrics on `:9090`
- No bootstrap peers (isolated node)

### Run tests

```bash
make test          # all tests with -race (120s timeout)
make test-unit     # unit tests only (fast, no network)
make test-int      # integration tests (libp2p nodes in-process)
make lint          # golangci-lint
```

### Observability stack

```bash
make compose-up    # p2p-server + Prometheus + Grafana
make compose-logs  # follow logs
make compose-down  # tear down
```

- **Grafana**: http://localhost:3000 (no login, anonymous admin)
- **Prometheus**: http://localhost:9091
- **Health check**: http://localhost:8080/healthz
- **Metrics**: http://localhost:9090/metrics

## Project Structure

```
cmd/server/              Entry point (main.go)
internal/
  config/                Viper config loader (dev/staging/prod YAML)
  node/                  Identity, host builder, Node orchestrator
  dht/                   Dual DHT (WAN+LAN) with bootstrap
  relay/                 Circuit Relay v2 service and readiness check
  metrics/               Prometheus collector + handler
  health/                /healthz and /readyz handlers
  autonat/               AutoNAT service stub
config/                  Environment YAML files (dev, staging, prod)
deployments/
  docker/                Dockerfile, docker-compose, Prometheus, Grafana
  terraform/             AWS infrastructure (IaC)
scripts/
  deploy.sh              SSH deploy to EC2
  gen_identity/          CLI tool for AWS Secrets Manager key generation
.github/workflows/       CI (lint+test+security), Build & Deploy, Production deploy
```

## Configuration

Configuration is loaded from `config/config.{APP_ENV}.yaml` via Viper. Environment variables override YAML with the `P2P_` prefix.

| Parameter | Dev | Staging/Prod |
|---|---|---|
| Identity source | Local file (`.local/`) | AWS Secrets Manager |
| Bootstrap peers | None | IPFS DHT defaults |
| Log level | `debug` | `debug` / `info` |
| Log format | `pretty` | `json` |

Key environment variables:

```bash
APP_ENV=dev                    # dev | staging | production
P2P_LOGGING_LEVEL=debug       # debug | info | warn | error
P2P_LOGGING_FORMAT=pretty     # pretty | json
P2P_P2P_LISTEN_TCP=/ip4/0.0.0.0/tcp/4001
P2P_P2P_LISTEN_QUIC=/ip4/0.0.0.0/udp/4001/quic-v1
```

## Architecture

The server is **fully stateless** — no database, all state in RAM. Single binary, designed for a `t4g.nano` EC2 instance.

```
Internet
  |
  v
[TCP:4001 + QUIC:4001]  libp2p host (Ed25519 identity)
  |-- Dual DHT           Kademlia peer discovery (WAN + LAN)
  |-- Circuit Relay v2   NAT traversal fallback (128 reservations, 64 circuits)
  |-- AutoNAT            Reachability probing for peers
  |-- Identify           Peer agent version exchange
  |
[HTTP:8080]              Health server (/healthz, /readyz)
[HTTP:9090]              Prometheus metrics (/metrics)
```

Startup sequence: config > logger > identity > host+DHT > collector > health server > metrics server > signal wait > graceful shutdown.

See [architecture.md](architecture.md) for the full technical design and ADRs.

## CI/CD

| Trigger | Workflow | Pipeline |
|---|---|---|
| PR to main | `ci.yml` | lint + test (80% gate) + govulncheck |
| Push to main | `build.yml` | test > build ARM64 > push ECR > deploy staging > Slack |
| Manual dispatch | `deploy.yml` | confirm "PRODUCCION" > environment approval > deploy prod > Slack |

AWS authentication uses **OIDC** (no static credentials). See [roadmap.md](roadmap.md) for the full development plan.

## Makefile Targets

```
make help              Show all targets
make dev               Run server locally (dev mode)
make test              All tests with -race
make lint              golangci-lint
make build             Docker image (linux/arm64)
make compose-up        Observability stack (p2p + Prometheus + Grafana)
make compose-down      Stop stack
make healthcheck       Verify /healthz and /readyz
make gen-identity      Generate Ed25519 key in AWS Secrets Manager
```

## Ports

| Port | Protocol | Service |
|---|---|---|
| 4001 | TCP + UDP (QUIC) | libp2p host |
| 8080 | HTTP | Health endpoints |
| 9090 | HTTP | Prometheus metrics |
