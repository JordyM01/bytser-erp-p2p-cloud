# Bytser ERP-P2P-CLOUD — Documento de Arquitectura

**Versión:** 1.1  
**Estado:** ✅ APROBADO  
**Fecha:** Marzo 2026  
**Repositorio:** `bytsers/erp-p2p-cloud` (independiente)  
**Stack:** Go + go-libp2p  
**Dominio:**  — cubierto por wildcard ACM  (AWS Certificate Manager)  

> **Aviso de alcance:** Este documento describe **únicamente** el servidor ERP-P2P-CLOUD. Este proyecto NO es el hub de sincronización CRDT, NO gestiona licencias u organizaciones, y NO expone una API de administración. Esas responsabilidades pertenecen al proyecto ERP-CLOUD-SYNC, que es un repositorio separado.

---

## Índice

1. [Propósito y Alcance](#1-propósito-y-alcance)
2. [Qué es y qué NO es este proyecto](#2-qué-es-y-qué-no-es-este-proyecto)
3. [Justificación de la Arquitectura Stateless](#3-justificación-de-la-arquitectura-stateless)
4. [Framework ICE — Jerarquía de Conexión](#4-framework-ice--jerarquía-de-conexión)
5. [Arquitectura del Nodo go-libp2p](#5-arquitectura-del-nodo-go-libp2p)
6. [Árbol de Directorios](#6-árbol-de-directorios)
7. [Modelo de Concurrencia](#7-modelo-de-concurrencia)
8. [Stack de Observabilidad](#8-stack-de-observabilidad)
9. [Seguridad del Nodo Relay](#9-seguridad-del-nodo-relay)
10. [Infraestructura e IaC](#10-infraestructura-e-iac)
11. [Pipeline CI/CD](#11-pipeline-cicd)
12. [Estrategia de Pruebas](#12-estrategia-de-pruebas)
13. [Gestión de Configuración](#13-gestión-de-configuración)
14. [Runbook Operativo](#14-runbook-operativo)
15. [Mapa de Dependencias](#15-mapa-de-dependencias)
16. [ADRs — Registros de Decisiones de Arquitectura](#16-adrs--registros-de-decisiones-de-arquitectura)

---

## 1. Propósito y Alcance

**Bytser ERP-P2P-CLOUD** existe para resolver un único problema: **permitir que dos nodos ERP-CORE detrás de routers y firewalls distintos se encuentren en internet y establezcan una conexión directa Peer-to-Peer (P2P), al menor costo posible de infraestructura.**

Este servidor implementa el framework **ICE (Interactive Connectivity Establishment)**, que es el estándar de oro en la industria para establecimiento de conectividad en redes descentralizadas. Ejecuta una escalera de tres intentos en orden creciente de costo, garantizando que el tráfico de datos pesado fluya directamente entre los dispositivos de los clientes siempre que sea técnicamente posible, y solo triangula por la nube cuando es estrictamente necesario.

```
┌────────────────────────────────────────────────────────────────────────┐
│                    BYTSER ERP-P2P-CLOUD                                 │
│                    relay.bytsers.com:4001 (TCP + QUIC/UDP)              │
│                                                                         │
│  Responsabilidad única:                                                 │
│  Ayudar a que el Nodo A y el Nodo B se encuentren y se conecten.       │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Fase 1: Señalización (Signaling Server)                         │  │
│  │  → Directorio de encuentro inicial vía WebSockets efímeros       │  │
│  ├──────────────────────────────────────────────────────────────────┤  │
│  │  Fase 2: STUN + UDP Hole Punching (DCUtR)                        │  │
│  │  → Perforar firewalls para conexión directa (80–90% de casos)    │  │
│  ├──────────────────────────────────────────────────────────────────┤  │
│  │  Fase 3: Circuit Relay v2 / TURN (respaldo)                      │  │
│  │  → Triangular tráfico solo cuando lo directo es imposible (10–20%)│  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  Estado: NINGUNO en disco. Solo RAM.                                    │
│  Base de datos: NO TIENE. No necesita.                                  │
│  Costo: t4g.nano (~$3.50/mes) + IP Elástica (~$3.60/mes) = ~$7/mes    │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Qué es y qué NO es este proyecto

### ✅ Esto SÍ hace ERP-P2P-CLOUD

| Función | Descripción |
|---|---|
| **Servidor de señalización** | Punto de encuentro inicial donde los nodos intercambian su PeerID, IPs públicas y puertos vía WebSockets efímeros |
| **AutoNAT / STUN** | Informa a cada nodo cómo se ve desde internet ("tu IP pública es X.X.X.X") |
| **Coordinador DCUtR (Hole Punching)** | Orquesta el intento simultáneo de perforación de NAT para lograr conexión directa UDP |
| **Bootstrap DHT** | Actúa como punto de entrada conocido a la red Kademlia para que los nodos se descubran |
| **Circuit Relay v2 (TURN)** | Triangula tráfico para el ~10–20% de nodos detrás de NAT simétrico o firewalls estrictos |
| **Endpoint /metrics** | Expone métricas operativas en RAM (formato Prometheus) sin escribir en disco |

### ❌ Esto NO hace ERP-P2P-CLOUD

| Lo que NO hace | Dónde vive esa responsabilidad |
|---|---|
| Sincronizar changesets CRDT | ERP-CLOUD-SYNC |
| Gestionar organizaciones o usuarios | ERP-CLOUD-SYNC |
| Emitir o validar licencias | ERP-CLOUD-SYNC |
| Distribuir plugins `.wasm` | ERP-CLOUD-SYNC |
| Autenticar dispositivos con JWT o mTLS | ERP-CLOUD-SYNC |
| Almacenar datos de negocio | ERP-CLOUD-SYNC / ERP-CORE |
| Exponer API de administración | ERP-CLOUD-SYNC / ERP-ADMIN-WEB |
| Conocer el contenido de los paquetes que reenvía | Nadie — el relay es ciego y los paquetes viajan cifrados |

---

## 3. Justificación de la Arquitectura Stateless

### 3.1 El servidor P2P es un "Cartero Ciego"

El servidor P2P no sabe qué contienen los paquetes que enruta, ni quién es el usuario, ni qué empresa opera el nodo. Su única función es conectar puntos en la red. Por diseño, esta ignorancia es una ventaja de seguridad y de costo.

```
┌──────────────────────────────────────────────────────────────────┐
│               COMPARACIÓN: STATEFUL vs STATELESS                  │
├──────────────────────────┬───────────────────────────────────────┤
│   ERP-CLOUD-SYNC         │   ERP-P2P-CLOUD (este proyecto)       │
│   (proyecto separado)    │                                        │
├──────────────────────────┼───────────────────────────────────────┤
│ PostgreSQL: historiales, │ PostgreSQL: NO EXISTE                  │
│ licencias, changesets    │                                        │
├──────────────────────────┼───────────────────────────────────────┤
│ Procesa lógica de        │ Procesa paquetes UDP en microsegundos  │
│ negocio compleja         │ sin lógica de negocio                  │
├──────────────────────────┼───────────────────────────────────────┤
│ Stateful — datos         │ Stateless — solo RAM                   │
│ persisten                │ (DHT, slots de relay, PeerIDs)         │
├──────────────────────────┼───────────────────────────────────────┤
│ t4g.large (~$48/mes)     │ t4g.nano (~$3.50/mes)                  │
└──────────────────────────┴───────────────────────────────────────┘
```

### 3.2 Por qué NO debe tener base de datos

**Razón 1 — Naturaleza volátil de los datos:**  
Un registro DHT que expira no es una pérdida de datos. Los PeerIDs, IPs y slots de relay son efímeros por diseño: si el servidor reinicia, los nodos ERP-CORE se reconectan automáticamente en segundos y reconstruyen la tabla de enrutamiento. No hay nada que persistir.

**Razón 2 — Evitar I/O blocking:**  
Para procesar miles de pequeños paquetes UDP por segundo, cada goroutine debe completarse en microsegundos. Una query a PostgreSQL introduce latencia de red interna (mínimo ~1ms) que degradaría la capacidad de enrutamiento. El servidor dedica el 100% de sus recursos a procesar tráfico de red.

**Razón 3 — Escalabilidad horizontal instantánea:**  
Al no tener estado compartido en disco, se puede clonar la instancia y crear un Auto Scaling Group en minutos sin sincronizar bases de datos entre réplicas. Cada instancia es idéntica e independiente.

**Razón 4 — Separación de responsabilidades:**  
El ERP-CLOUD-SYNC ya gestiona todos los datos persistentes del ecosistema. Añadir una base de datos aquí crearía duplicación de responsabilidades y acoplamiento innecesario entre proyectos.

### 3.3 Almacenamiento en RAM únicamente

```
Estado del proceso ERP-P2P-CLOUD en tiempo de ejecución:

┌──────────────────────────────────────────────────┐
│                    RAM del proceso                 │
│                                                    │
│  Kademlia DHT Routing Table                        │
│  ┌─────────────────────────────────────────────┐  │
│  │ PeerID → [multiaddr1, multiaddr2, ...]      │  │
│  │ (mapa en memoria, reconstruible en segundos)│  │
│  └─────────────────────────────────────────────┘  │
│                                                    │
│  Circuit Relay v2 Slots                            │
│  ┌─────────────────────────────────────────────┐  │
│  │ max 128 reservaciones activas               │  │
│  │ max 64 circuitos simultáneos                │  │
│  │ TTL: 1h, auto-expiración                    │  │
│  └─────────────────────────────────────────────┘  │
│                                                    │
│  Métricas Prometheus (contadores/gauges)           │
│  ┌─────────────────────────────────────────────┐  │
│  │ p2p_active_websocket_connections            │  │
│  │ p2p_turn_bytes_relayed_total                │  │
│  │ p2p_relay_circuits_active                   │  │
│  │ ... (ver Sección 8)                         │  │
│  └─────────────────────────────────────────────┘  │
│                                                    │
│  Identidad Ed25519 del nodo (PeerID estable)       │
│  ┌─────────────────────────────────────────────┐  │
│  │ Cargada desde AWS Secrets Manager al inicio │  │
│  │ PeerID = hash(clave pública Ed25519)         │  │
│  │ Estable entre reinicios (keypair persistido) │  │
│  └─────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘

DISCO: Solo el binario Go compilado. Nada más.
```

---

## 4. Framework ICE — Jerarquía de Conexión

Cuando un nodo ERP-CORE arranca y quiere conectarse con sus pares, ejecuta automáticamente esta escalera en orden, deteniéndose en la primera que tenga éxito.

```
┌─────────────────────────────────────────────────────────────────────┐
│              ESCALERA DE CONEXIÓN ICE                                │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ INTENTO 1 — mDNS Local (LAN)                                   │ │
│  │ Costo: $0 — el servidor P2P ni se entera                       │ │
│  │                                                                  │ │
│  │ ERP-Core transmite consulta mDNS en su red local (WiFi/Ethernet)│ │
│  │ Si el peer está en la misma red → conexión directa por IP LAN  │ │
│  │ Latencia: <5ms                                                  │ │
│  │ Casos: todos los nodos en la misma oficina/sucursal             │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                          │ Si falla (peer en otra red)               │
│                          ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ INTENTO 2 — Señalización + STUN + Hole Punching (WAN directo)  │ │
│  │ Costo de ancho de banda: $0 (solo señalización en ms)          │ │
│  │                                                                  │ │
│  │ 1. Nodo se conecta a relay.bytsers.com vía WebSocket            │ │
│  │ 2. AutoNAT: servidor informa al nodo su IP pública             │ │
│  │ 3. Nodo publica su PeerID + multiaddrs en el DHT Kademlia      │ │
│  │ 4. Nodo descubre PeerIDs de su misma organización en el DHT    │ │
│  │ 5. DCUtR: ambos nodos coordinan hole-punch UDP simultáneo      │ │
│  │ 6. Si exitoso → conexión directa (relay se cierra)             │ │
│  │ Tasa de éxito: 80–90%                                          │ │
│  │ Latencia post-conexión: 20–80ms (según distancia geográfica)   │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                          │ Si falla (NAT simétrico, firewall estricto)│
│                          ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ INTENTO 3 — Circuit Relay v2 / TURN (respaldo)                 │ │
│  │ Costo: ancho de banda AWS (solo ~10–20% de usuarios)           │ │
│  │                                                                  │ │
│  │ Nodo reserva un slot en el Circuit Relay v2 del servidor       │ │
│  │ Tráfico viaja: NodoA → relay.bytsers.com → NodoB                │ │
│  │ Limitado: 128KB/circuito, 5 min máx por circuito              │ │
│  │ Propósito: handshake inicial y bootstrapping de sync           │ │
│  │ Métrica de costo: p2p_turn_bytes_relayed_total (ver Sección 8) │ │
│  │ Casos: firewalls corporativos muy estrictos, redes celulares   │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.1 Diagrama de Secuencia: DCUtR Hole Punching

```
ERP-Core A          relay.bytsers.com         ERP-Core B
(detrás de NAT)     (este servidor)          (detrás de NAT)
     │                     │                      │
     │──── WebSocket ──────►│                      │
     │◄─── "tu IP: A.A.A.A"─│                      │
     │                     │◄──── WebSocket ───────│
     │                     │─── "tu IP: B.B.B.B" ─►│
     │                     │                      │
     │──── PUT DHT ─────────►│                      │
     │                     │◄───── PUT DHT ────────│
     │                     │                      │
     │◄─── GET DHT ─────────│──── GET DHT ─────────►│
     │   (encuentra B)      │   (encuentra A)      │
     │                     │                      │
     │═══════ Fase DCUtR: ambos perforan NAT simultáneamente ════════│
     │                                                                │
     │◄══════════════ CONEXIÓN UDP DIRECTA ══════════════════════════►│
     │                  (relay se cierra)                             │
     │                     │                      │
     │      ╔═══════════════╧═══════════════╗      │
     │      ║ Si hole-punch falla:          ║      │
     │      ║ Ambos reservan slot relay     ║      │
     │      ║ A ──► relay ──► B             ║      │
     │      ╚═══════════════════════════════╝      │
```

---

## 5. Arquitectura del Nodo go-libp2p

### 5.1 Componentes del Host

El servidor es un único binario Go que construye un host go-libp2p con los siguientes behaviours habilitados:

```
┌──────────────────────────────────────────────────────────────────┐
│                    HOST go-libp2p                                  │
│                    PeerID: estable (desde Secrets Manager)         │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    TRANSPORTES                                │ │
│  │  QUIC (primario)          TCP + Noise + Yamux (fallback)     │ │
│  │  /udp/4001/quic-v1        /tcp/4001/noise/yamux             │ │
│  │  TLS 1.3 nativo           AES-256-GCM cifrado               │ │
│  │  0-RTT connection         Sin head-of-line blocking          │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │ Kademlia DHT │  │ Circuit      │  │ AutoNAT                  │ │
│  │ (server mode)│  │ Relay v2     │  │                          │ │
│  │              │  │              │  │ Responde sondeos:        │ │
│  │ Namespace:   │  │ Max 128 rsv. │  │ "¿soy alcanzable?"       │ │
│  │ /bytser/erp  │  │ Max 64 circ. │  │                          │ │
│  │              │  │ TTL: 1h      │  │ Informa IP pública       │ │
│  │ Tabla en RAM │  │ Max 128KB    │  │ a cada nodo conectado    │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
│                                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │ Identify     │  │ DCUtR        │  │ Ping                     │ │
│  │              │  │              │  │                          │ │
│  │ Intercambia  │  │ Coordina     │  │ Keep-alive entre         │ │
│  │ metadata con │  │ hole-punch   │  │ peers conectados         │ │
│  │ peers        │  │ para upgrade │  │                          │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### 5.2 Secuencia de Inicio del Servidor

```go
// Orden de inicialización en main.go

1. Cargar configuración (Viper: YAML + env vars)
2. Inicializar logger estructurado (zerolog, stdout → CloudWatch)
3. Cargar keypair Ed25519 desde AWS Secrets Manager
   → derivar PeerID estable (mismo en todos los reinicios)
4. Construir host go-libp2p:
   a. Configurar transportes: QUIC (primario) + TCP+Noise+Yamux (fallback)
   b. Bind a /ip4/0.0.0.0/tcp/4001 y /ip4/0.0.0.0/udp/4001/quic-v1
5. Iniciar Kademlia DHT en modo servidor (namespace /bytser/erp)
6. Habilitar Circuit Relay v2 (limits: 128 rsv, 64 circuits, TTL 1h)
7. Habilitar AutoNAT service
8. Habilitar Identify protocol
9. Iniciar servidor HTTP en :9090 → endpoint /metrics (Prometheus)
10. Iniciar servidor HTTP en :8080 → endpoint /healthz + /readyz
11. Registrar manejador de señales OS (SIGTERM, SIGINT) para graceful shutdown
12. Loguear: "ERP-P2P-CLOUD listo. PeerID: 12D3KooW..."
13. Bloquear en event loop del host (procesar eventos de red indefinidamente)
```

### 5.3 Identidad del Servidor (PeerID Estable)

La estabilidad del PeerID es crítica: este valor se distribuye a todos los nodos ERP-CORE como la dirección de bootstrap. Si cambia, los nodos pierden el punto de entrada a la red.

```
Primera ejecución (o después de rotar credenciales):
  → AWS Secrets Manager crea/retorna keypair Ed25519
  → PeerID = base58(sha256(clave_pública))
  → PeerID se registra en la documentación y en la config de ERP-CORE

Reinicios subsecuentes:
  → El mismo keypair se carga desde Secrets Manager
  → El mismo PeerID se deriva
  → Los nodos ERP-CORE siguen conectando al mismo bootstrap peer

Multiaddr completo del bootstrap peer:
  /dns4/relay.bytsers.com/tcp/4001/p2p/12D3KooW[PEER_ID]
  /dns4/relay.bytsers.com/udp/4001/quic-v1/p2p/12D3KooW[PEER_ID]
```

---

## 6. Árbol de Directorios

```
bytsers-erp-p2p-cloud/
│
├── cmd/
│   └── server/
│       └── main.go              # Entry point: construcción del host y arranque
│
├── internal/
│   ├── node/
│   │   ├── host.go              # Construcción del host go-libp2p y todos sus behaviours
│   │   ├── host_test.go
│   │   ├── identity.go          # Carga/generación del keypair Ed25519
│   │   └── identity_test.go
│   │
│   ├── dht/
│   │   ├── bootstrap.go         # Kademlia DHT en modo servidor, namespace /bytser/erp
│   │   └── bootstrap_test.go
│   │
│   ├── relay/
│   │   ├── service.go           # Circuit Relay v2: límites, TTL, manejo de slots
│   │   └── service_test.go
│   │
│   ├── autonat/
│   │   ├── service.go           # Servicio AutoNAT: responder sondeos de peers
│   │   └── service_test.go
│   │
│   ├── metrics/
│   │   ├── collector.go         # Definición y registro de métricas Prometheus (en RAM)
│   │   ├── collector_test.go
│   │   └── handler.go           # Handler HTTP para /metrics (formato Prometheus)
│   │
│   ├── health/
│   │   ├── handler.go           # Handlers HTTP para /healthz y /readyz
│   │   └── handler_test.go
│   │
│   └── config/
│       ├── config.go            # Struct de configuración + loader (Viper)
│       └── config_test.go
│
├── deployments/
│   ├── docker/
│   │   └── Dockerfile           # Multi-stage build, imagen final ~15MB (alpine)
│   └── terraform/
│       ├── main.tf              # Provider AWS, backend S3
│       ├── variables.tf
│       ├── outputs.tf
│       ├── ec2.tf               # Instancia t4g.nano, user-data bootstrap
│       ├── vpc.tf               # VPC, subnet pública, IGW
│       ├── security_groups.tf   # Puertos 4001/TCP, 4001/UDP, 9090 (interno), 8080 (interno)
│       ├── route53.tf           # Registro A: relay.bytsers.com
│       ├── iam.tf               # Rol EC2: solo Secrets Manager read + CloudWatch write
│       └── cloudwatch.tf        # Grupo de logs /bytsers/p2p-cloud, alarma CPU > 80%
│
├── scripts/
│   ├── generate_identity.sh     # Script one-time para generar y guardar keypair en Secrets Manager
│   └── healthcheck.sh
│
├── .github/
│   └── workflows/
│       ├── ci.yml               # lint + test en PR
│       ├── build.yml            # build + push imagen Docker en push a main
│       └── deploy.yml           # deploy a EC2 (manual, requiere aprobación)
│
├── config/
│   ├── config.dev.yaml
│   ├── config.staging.yaml
│   └── config.prod.yaml
│
├── Makefile
├── .golangci.yml
├── go.mod
├── go.sum
└── README.md
```

**Nota sobre la simplicidad:** Este proyecto es intencionalmente pequeño. No hay `internal/repository/`, no hay `db/migrations/`, no hay `proto/`, no hay `gen/`. La ausencia de estas carpetas no es un olvido — es el diseño correcto para un servidor stateless.

---

## 7. Modelo de Concurrencia

### 7.1 Arquitectura de Goroutines

```
goroutine principal (main)
│
├── goroutine: event loop del host go-libp2p
│   │
│   ├── goroutines por conexión entrante (una por peer conectado)
│   │   └── protocolo de mensajería (Identify, Ping, AutoNAT probe, etc.)
│   │
│   ├── goroutines de circuitos relay activos (máx 64 simultáneas)
│   │   └── forwarding de paquetes: NodoA ↔ relay ↔ NodoB
│   │
│   └── goroutine DHT: mantenimiento de tabla Kademlia (refresh, expire)
│
├── goroutine: servidor HTTP /metrics + /healthz + /readyz (puerto 9090/8080)
│
├── goroutine: colector de métricas internas (ticker: 15s)
│   └── actualiza gauges: peers_connected, relay_circuits_active, etc.
│
└── goroutine: graceful shutdown coordinator
    └── escucha SIGTERM/SIGINT → drena conexiones → exit 0
```

### 7.2 Manejo de Conexiones

```go
// Patrón central de manejo de conexiones en go-libp2p

// go-libp2p maneja internamente un pool de goroutines por conexión.
// El servidor solo define los límites de recursos:

host, _ := libp2p.New(
    libp2p.ListenAddrStrings(
        "/ip4/0.0.0.0/tcp/4001",
        "/ip4/0.0.0.0/udp/4001/quic-v1",
    ),
    libp2p.ResourceManager(network.NewResourceManager(
        rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits), // configurar límites reales
    )),
    // Límites de conexión para proteger contra DoS:
    libp2p.ConnectionManager(connmgr.NewConnManager(
        900,  // low water: comenzar a cerrar conexiones inactivas
        1000, // high water: límite máximo de peers simultáneos
        connmgr.WithGracePeriod(30 * time.Second),
    )),
)
```

### 7.3 Graceful Shutdown

```
SIGTERM recibido
    │
    ├── Dejar de aceptar nuevas conexiones relay
    ├── Notificar peers conectados (libp2p graceful disconnect)
    ├── Esperar a que los circuitos relay activos completen o expiren (timeout 30s)
    ├── Cerrar el host go-libp2p (host.Close())
    ├── Cerrar servidor HTTP de métricas
    └── Exit 0
```

---

## 8. Stack de Observabilidad

El servidor P2P es stateless, pero el ecosistema Bytser mantiene visibilidad total sobre el estado de la red mediante cuatro niveles de observabilidad.

```
┌─────────────────────────────────────────────────────────────────────┐
│              CUATRO NIVELES DE OBSERVABILIDAD DEL ECOSISTEMA         │
│                                                                      │
│  Nivel 1 ── ERP-CORE (Rust) — diagnóstico local en el dispositivo   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ crate tracing → logs estructurados locales con rotación      │   │
│  │ Niveles: ERROR, WARN, INFO, DEBUG, TRACE                     │   │
│  │ Captura: intentos fallidos DCUtR, bloqueos mDNS, errores     │   │
│  │ libp2p nativos (ej: "antivirus bloqueó UDP hole-punch")      │   │
│  │ → Herramienta de soporte técnico de campo                    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Nivel 2 ── ERP-CLOUD-SYNC — detección de anomalías P2P             │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Si la Caja 3 pierde P2P → sube todo al Cloud como respaldo   │   │
│  │ Grafana detecta pico en:                                     │   │
│  │   erp_cloud_sync_sync_operations_pushed_total{device=caja3}  │   │
│  │ mientras otras cajas mantienen tráfico Cloud ≈ 0            │   │
│  │ → Alerta automática: "Caja 3 aislada de su red P2P local"   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Nivel 3 ── ERP-P2P-CLOUD (este servidor) — métricas del relay      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Endpoint /metrics (Prometheus, puerto 9090, solo red interna)│   │
│  │ Datos en RAM únicamente — NADA se escribe en disco           │   │
│  │ Enviadas al Grafana centralizado del ERP-CLOUD-SYNC          │   │
│  │                                                              │   │
│  │ MÉTRICAS CLAVE:                                              │   │
│  │  p2p_active_websocket_connections  → usuarios en señalización│   │
│  │  p2p_turn_bytes_relayed_total      → MBs triangulados        │   │
│  │                                     (métrica de costo AWS)   │   │
│  │  p2p_relay_circuits_active         → circuitos TURN activos  │   │
│  │  p2p_relay_reservations_active     → slots reservados        │   │
│  │  p2p_dht_peers_in_routing_table    → salud de la red DHT     │   │
│  │  p2p_hole_punch_attempts_total     → intentos hole-punch     │   │
│  │  p2p_hole_punch_success_total      → hole-punches exitosos   │   │
│  │  p2p_autonat_reachable_total       → nodos con IP pública    │   │
│  │  p2p_autonat_nat_total             → nodos detrás de NAT     │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Nivel 4 ── ERP-Desktop (Flutter) — trazabilidad visual del usuario │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ SyncStatusWidget en AppBar, alimentado por SSE desde ERP-CORE│   │
│  │ Estados en tiempo real:                                      │   │
│  │   🟢 P2P LAN activo (conexión directa en la misma red)       │   │
│  │   🔵 Triangulando por nube (usando Cloud Sync como respaldo)  │   │
│  │   🟡 Relay P2P activo (Circuit Relay v2 en uso)              │   │
│  │   🔴 Modo Offline (sin conectividad a ningún peer ni nube)   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 8.1 Logging Estructurado (zerolog)

Cada entrada de log incluye:

```json
{
  "level": "info",
  "time": "2026-03-18T10:00:00Z",
  "service": "bytsers-erp-p2p-cloud",
  "version": "1.0.0",
  "peer_id": "12D3KooW...",
  "event": "relay_circuit_opened",
  "remote_peer": "12D3KooW...",
  "circuit_id": "uuid",
  "duration_ms": 0,
  "message": "circuito relay establecido"
}
```

Niveles:

- `DEBUG`: eventos internos del host go-libp2p (deshabilitado en producción)
- `INFO`: conexiones, circuitos abiertos/cerrados, resultados de hole-punch
- `WARN`: límites de slots relay al 80%, picos de bytes triangulados
- `ERROR`: fallos de inicialización, errores de carga de identidad
- `FATAL`: incapacidad de bind en puertos o cargar keypair al inicio

Salida: stdout → agente CloudWatch Logs en EC2 → grupo `/bytsers/p2p-cloud`

### 8.2 Health Checks

- `GET /healthz` — siempre retorna `200 OK` si el proceso está vivo
- `GET /readyz` — retorna `200 OK` solo si:
  - El host go-libp2p está escuchando en al menos un transporte
  - El DHT está inicializado
  - El módulo Circuit Relay v2 está activo

---

## 9. Seguridad del Nodo Relay

### 9.1 El relay no conoce el contenido

Todos los paquetes que el servidor enruta viajan cifrados de extremo a extremo por los transportes de go-libp2p (Noise Protocol sobre TCP, TLS 1.3 sobre QUIC). El servidor es un enrutador ciego — físicamente no puede leer el contenido de lo que reenvía.

### 9.2 Protección contra abuso del relay

```
Mecanismo                          Límite
─────────────────────────────────────────────────────────
max_reservations                   128 slots simultáneos
max_circuits                       64 circuitos activos
reservation_ttl                    1 hora
max_circuit_duration               5 minutos
max_circuit_bytes                  128 KB por circuito
throttle_autonat_per_peer          1 sondeo cada 30 segundos
connection_manager_high_watermark  1000 peers simultáneos
```

Estos límites protegen contra:

- **DoS por saturación de slots:** un atacante no puede reservar todos los slots del relay
- **Abuso de ancho de banda:** los 128KB por circuito son suficientes para el handshake P2P, pero no para transferir datos de negocio voluminosos (que deben ir por conexión directa)
- **Peers fantasma:** el TTL de 1h garantiza que los slots expirados se liberan automáticamente

### 9.3 Identidad y autenticidad del servidor

La identidad Ed25519 del servidor (almacenada en AWS Secrets Manager) garantiza que:

- Los nodos ERP-CORE se conectan al servidor auténtico (el PeerID está hardcodeado en su config de bootstrap)
- Un atacante no puede suplantar el relay server sin acceso a la clave privada
- Las conexiones entre peers siempre usan cifrado Noise/TLS, incluso cuando el tráfico pasa por el relay

---

## 10. Infraestructura e IaC

### 10.1 Instancia EC2

```
Entorno       Instancia     vCPU   RAM    Costo/mes   Notas
──────────────────────────────────────────────────────────────────
Dev/Staging   t4g.nano      2      0.5GB  ~$3.50      Suficiente para pruebas
Producción    t4g.nano      2      0.5GB  ~$3.50      Suficiente hasta 500+ nodos
              t4g.micro     2      1GB    ~$7.50      Si CPU > 70% sostenido
IP Elástica   (fija)        —      —      ~$3.60      Necesaria para PeerID estable
─────────────────────────────────────────────────────────────────
Total producción (t4g.nano): ~$7.10/mes
```

**Por qué t4g.nano es suficiente:**  
El binario Go compilado para linux/arm64 usa ~30–50MB de RAM en reposo. go-libp2p es eficiente en memoria. La tabla DHT con 1000 peers ocupa ~5MB adicionales. El servidor vive y muere por CPU para procesar paquetes UDP — y las instancias t4g tienen performance de red excelente para su precio.

### 10.2 Dockerfile (Multi-Stage)

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags="-w -s" -o /erp-p2p-cloud ./cmd/server

# Stage 2: Imagen final (~15MB)
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /erp-p2p-cloud .
EXPOSE 4001/tcp 4001/udp 8080 9090
ENTRYPOINT ["/app/erp-p2p-cloud"]
```

No hay `docker-compose.yml` de producción con múltiples servicios. Este servidor corre solo. No tiene PostgreSQL ni Redis como dependencias en producción.

```yaml
# docker-compose.yml solo existe para desarrollo local
# únicamente levanta el binario con hot-reload
services:
  p2p-server:
    build: .
    ports:
      - "4001:4001/tcp"
      - "4001:4001/udp"
      - "8080:8080"
      - "9090:9090"
    environment:
      - APP_ENV=dev
      - LOG_LEVEL=debug
    # En dev, identidad se genera localmente (sin Secrets Manager)
```

### 10.3 Estructura Terraform

```
deployments/terraform/

main.tf              → provider aws, backend S3+DynamoDB para lock de estado
variables.tf         → region, instance_type, keypair_name, domain_name
outputs.tf           → public_ip, instance_id, relay_multiaddr

vpc.tf               → VPC mínima:
                         subnet pública (la instancia tiene IP pública)
                         sin subnet privada (no hay RDS, no hay Redis)
                         Internet Gateway

security_groups.tf   → sg-p2p-relay:
                         INGRESS 4001/tcp  0.0.0.0/0  (peers TCP)
                         INGRESS 4001/udp  0.0.0.0/0  (peers QUIC)
                         INGRESS 22/tcp    [IP_BASTION] (SSH solo desde bastion)
                         EGRESS  ALL       0.0.0.0/0
                       sg-p2p-internal (solo desde dentro de la VPC):
                         INGRESS 8080/tcp  VPC CIDR    (/healthz, /readyz)
                         INGRESS 9090/tcp  VPC CIDR    (/metrics Prometheus)

ec2.tf               → instancia t4g.nano:
                         ami: ubuntu-24.04-arm64
                         instance_profile: rol IAM con solo Secrets Manager + CloudWatch
                         user_data: instala Docker, arranca contenedor erp-p2p-cloud
                         eip_association: IP Elástica fija

route53.tf           → registro A: relay.bytsers.com → EIP

iam.tf               → rol EC2 con políticas mínimas:
                         secretsmanager:GetSecretValue (solo el secreto del keypair)
                         logs:CreateLogStream, logs:PutLogEvents (CloudWatch)

cloudwatch.tf        → grupo de logs: /bytsers/p2p-cloud (retención 30 días)
                       alarma: CPUUtilization > 80% por 5 min → SNS → email
```

### 10.4 Targets del Makefile

```makefile
.PHONY: help dev build push test lint deploy-staging deploy-prod logs

help:            ## Mostrar esta ayuda
dev:             ## Iniciar servidor en modo desarrollo (hot-reload)
build:           ## Construir imagen Docker linux/arm64
push:            ## Subir imagen a ECR
test:            ## Ejecutar todos los tests con -race
test-unit:       ## Solo tests unitarios
test-int:        ## Tests de integración (nodos libp2p en proceso)
lint:            ## Ejecutar golangci-lint
tf-plan:         ## terraform plan
tf-apply:        ## terraform apply
deploy-staging:  ## Desplegar a EC2 de staging
deploy-prod:     ## Desplegar a EC2 de producción (requiere aprobación)
logs:            ## Seguir logs de producción desde CloudWatch
healthcheck:     ## Verificar /healthz y /readyz
gen-identity:    ## Generar keypair Ed25519 y guardar en Secrets Manager (solo una vez)
```

---

## 11. Pipeline CI/CD

### 11.1 ci.yml (en cada PR)

```
Trigger: pull_request → main / develop

Jobs en paralelo:
├── lint
│   └── golangci-lint run ./...
│
├── test
│   ├── go test ./... -race -coverprofile=coverage.out
│   └── cobertura total debe ser > 80%
│
└── security
    ├── govulncheck ./...
    └── trivy image scan (solo si Dockerfile cambió)
```

### 11.2 build.yml (en push a main)

```
Trigger: push → main

Jobs secuenciales:
1. test (igual que ci.yml)
2. build
   ├── docker build --platform linux/arm64
   ├── tag: git SHA + semver (si el commit tiene tag)
   └── docker push a ECR
3. deploy-staging (automático)
   ├── SSH a EC2 staging
   ├── docker pull nueva imagen
   ├── docker stop/rm contenedor anterior
   ├── docker run nueva imagen
   └── curl /readyz (esperar healthy, timeout 30s)
```

### 11.3 deploy.yml (producción, manual)

```
Trigger: workflow_dispatch
Requiere: aprobación de 1 reviewer

Entradas:
- image_tag: tag de ECR a desplegar

Pasos:
1. Notificar Slack: "Deploy P2P relay iniciado: v{tag}"
2. SSH a EC2 producción
3. docker pull nueva imagen
4. docker stop/rm contenedor anterior (graceful: SIGTERM, 30s timeout)
5. docker run nueva imagen
6. curl /readyz (timeout 60s, retry cada 5s)
7. Notificar Slack: éxito o fallo
```

---

## 12. Estrategia de Pruebas

### 12.1 Enfoque

go-libp2p provee helpers de test excelentes para crear nodos en proceso. La mayor parte de los tests son tests de integración "en proceso" (no necesitan instancias externas):

```go
// Ejemplo: test de Circuit Relay v2 en proceso
func TestCircuitRelay_DosPeersDetrasDeNAT(t *testing.T) {
    // Crear relay server en proceso
    relayHost, _ := libp2p.New(libp2p.EnableRelayService())
    defer relayHost.Close()

    // Crear dos peers que solo conocen al relay (simulan estar detrás de NAT)
    peerA, _ := libp2p.New(
        libp2p.EnableAutoRelayWithStaticRelays(peer.AddrInfo{
            ID: relayHost.ID(), Addrs: relayHost.Addrs(),
        }),
    )
    peerB, _ := libp2p.New(
        libp2p.EnableAutoRelayWithStaticRelays(peer.AddrInfo{
            ID: relayHost.ID(), Addrs: relayHost.Addrs(),
        }),
    )
    defer peerA.Close()
    defer peerB.Close()

    // Verificar que A puede conectar a B a través del relay
    relayAddr := ma.StringCast(
        fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayHost.ID(), peerB.ID()),
    )
    err := peerA.Connect(ctx, peer.AddrInfo{ID: peerB.ID(), Addrs: []ma.Multiaddr{relayAddr}})
    assert.NoError(t, err)

    // Verificar que el slot de relay está ocupado
    assert.Equal(t, 1, getActiveCircuits(relayHost))
}
```

### 12.2 Cobertura por Paquete

| Paquete | Mínimo |
|---|---|
| `internal/node/` | 80% |
| `internal/dht/` | 75% |
| `internal/relay/` | 80% |
| `internal/autonat/` | 75% |
| `internal/metrics/` | 85% |
| `internal/health/` | 90% |
| `internal/config/` | 85% |
| **Total** | **80%** |

---

## 13. Gestión de Configuración

### 13.1 Struct de Configuración

```go
// internal/config/config.go
type Config struct {
    App     AppConfig
    Server  ServerConfig
    P2P     P2PConfig
    Metrics MetricsConfig
    Logging LoggingConfig
    AWS     AWSConfig
}

type AppConfig struct {
    Env     string // dev | staging | prod
    Version string
}

type ServerConfig struct {
    HealthPort  int    // default: 8080
    MetricsPort int    // default: 9090
}

type P2PConfig struct {
    // Transportes
    ListenTCP      string // default: /ip4/0.0.0.0/tcp/4001
    ListenQUIC     string // default: /ip4/0.0.0.0/udp/4001/quic-v1
    ExternalIP     string // IP pública del servidor (para anunciar en DHT)

    // DHT
    DHTNamespace   string // default: /bytser/erp

    // Circuit Relay v2
    RelayMaxReservations int           // default: 128
    RelayMaxCircuits     int           // default: 64
    RelayTTL             time.Duration // default: 1h
    RelayMaxCircuitDur   time.Duration // default: 5m
    RelayMaxCircuitBytes int64         // default: 131072 (128KB)

    // AutoNAT
    AutoNATEnabled       bool          // default: true
    AutoNATThrottlePeer  time.Duration // default: 30s

    // Connection Manager
    ConnMgrLowWater  int           // default: 900
    ConnMgrHighWater int           // default: 1000
    ConnMgrGrace     time.Duration // default: 30s

    // Identidad
    IdentitySecretName string // nombre del secreto en AWS Secrets Manager
}

type AWSConfig struct {
    Region          string
    SecretsEndpoint string // override para dev local (localstack)
}

type LoggingConfig struct {
    Level  string // debug | info | warn | error
    Format string // json | pretty
}
```

### 13.2 Prioridad de Carga

1. **AWS Secrets Manager** — keypair Ed25519 (el único "secreto" real)
2. **Variables de entorno** — overrides de cualquier campo del YAML
3. **Archivo YAML** — defaults por entorno

### 13.3 Configuraciones por Entorno

```yaml
# config/config.prod.yaml
app:
  env: production

server:
  health_port: 8080
  metrics_port: 9090

p2p:
  external_ip: ""            # detectado automáticamente vía AutoNAT si vacío
  dht_namespace: /bytser/erp
  relay_max_reservations: 128
  relay_max_circuits: 64
  relay_ttl: 1h
  relay_max_circuit_dur: 5m
  relay_max_circuit_bytes: 131072
  autonat_enabled: true
  conn_mgr_high_water: 1000
  identity_secret_name: bytsers/p2p-relay/identity

logging:
  level: info
  format: json

aws:
  region: us-east-1
```

---

## 14. Runbook Operativo

### 14.1 No hay backups

Este servidor no tiene datos que respaldar. Si la instancia se destruye completamente:

1. Terraform recrea la instancia en ~3 minutos
2. El keypair Ed25519 se carga desde AWS Secrets Manager (nunca se pierde)
3. Los nodos ERP-CORE se reconectan automáticamente al mismo PeerID en segundos
4. El DHT se reconstruye en RAM a medida que los nodos se conectan

**Tiempo de recuperación total ante falla completa: < 5 minutos.**

### 14.2 Alertas Operativas

| Alerta | Condición | Acción |
|---|---|---|
| CPU alto | CPUUtilization > 80% por 5 min (CloudWatch) | Revisar si relay está siendo abusado; considerar subir a t4g.micro |
| Slots relay saturados | `p2p_relay_reservations_active` > 110 (85% de 128) | Alerta en Grafana; evaluar segunda instancia relay |
| Alto tráfico TURN | `p2p_turn_bytes_relayed_total` crecimiento > 10GB/día | Revisar costos AWS; investigar si hay nodos abusando del relay |
| Tasa de hole-punch baja | `p2p_hole_punch_success_total / p2p_hole_punch_attempts_total` < 50% | Investigar configuración de red de los clientes afectados |
| Servidor caído | `/healthz` no responde por 2 min | Auto-restart del contenedor Docker (`restart: unless-stopped`) |

### 14.3 Rotación de Identidad (procedimiento excepcional)

```
⚠️  ADVERTENCIA: Rotar el keypair del relay cambia el PeerID.
    Todos los nodos ERP-CORE deben actualizar su config de bootstrap.
    Esto requiere un despliegue coordinado con el equipo de ERP-CORE.

Pasos:
1. Generar nuevo keypair: make gen-identity (guarda en Secrets Manager)
2. Actualizar config de bootstrap en ERP-CORE (nuevo PeerID)
3. Desplegar nueva versión de ERP-CORE a todos los clientes
4. Solo entonces: desplegar nueva versión de ERP-P2P-CLOUD
5. Monitorear reconexión de nodos en dashboard P2P de Grafana
```

---

## 15. Mapa de Dependencias

```
go.mod — dependencias de producción (mínimas intencionalmente):

Red P2P (núcleo del proyecto)
  github.com/libp2p/go-libp2p                    Host libp2p principal
  github.com/libp2p/go-libp2p-kad-dht            Kademlia DHT
  github.com/libp2p/go-libp2p/p2p/net/...        Connection manager, resource manager
  github.com/multiformats/go-multiaddr            Formato de direcciones libp2p

Observabilidad
  github.com/rs/zerolog                           Logging estructurado JSON
  github.com/prometheus/client_golang             Métricas Prometheus (en RAM)

AWS
  github.com/aws/aws-sdk-go-v2/service/secretsmanager  Cargar keypair Ed25519
  github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs  (opcional, vía log driver Docker)

Config
  github.com/spf13/viper                          Config YAML + env vars

Utilidades
  github.com/google/uuid                          IDs en logs

── Dependencias de desarrollo únicamente ──
  github.com/stretchr/testify                     Aserciones de tests
  go.uber.org/mock / github.com/stretchr/mock     Mocking en tests
```

**Lo que deliberadamente NO está en go.mod:**

- `github.com/jackc/pgx` — no hay base de datos
- `github.com/redis/go-redis` — no hay Redis
- `github.com/gin-gonic/gin` — el único servidor HTTP sirve `/metrics` y `/healthz`, se implementa con `net/http` de la stdlib
- `google.golang.org/grpc` — este servidor no expone API gRPC (los peers hablan libp2p, no gRPC)
- `github.com/golang-jwt/jwt` — no autentica usuarios
- `golang.org/x/crypto/bcrypt` — no hashea contraseñas

---

## 16. ADRs — Registros de Decisiones de Arquitectura

### ADR-001: Servidor Completamente Stateless, Sin Base de Datos

**Decisión:** El servidor ERP-P2P-CLOUD no tiene PostgreSQL, Redis ni ninguna base de datos.  
**Justificación:** Los datos del relay (PeerIDs, slots, tabla DHT) son volátiles por diseño. Persistirlos no añade valor porque los clientes los reconstruyen al reconectarse. Una base de datos introduciría I/O blocking que degradaría el procesamiento de paquetes UDP en tiempo real. La ausencia de estado permite escalar horizontalmente sin coordinación entre réplicas.  
**Consecuencia:** El tiempo de recuperación ante falla total es < 5 minutos — la instancia se recrea, carga su keypair de Secrets Manager, y los nodos se reconectan solos.

### ADR-002: go-libp2p sobre WebRTC Personalizado

**Decisión:** Usar go-libp2p para toda la funcionalidad de señalización, STUN, hole-punching y relay.  
**Justificación:** ERP-CORE usa rust-libp2p — la compatibilidad de protocolos es nativa y garantizada sin trabajo adicional. go-libp2p provee Circuit Relay v2, DCUtR, Kademlia DHT y AutoNAT como implementaciones battle-tested. Construir un stack WebRTC propio requeriría implementar STUN, TURN, ICE y señalización desde cero.

### ADR-003: Binario Único, Sin Docker Compose en Producción

**Decisión:** El servidor corre como un único contenedor Docker sin servicios auxiliares.  
**Justificación:** No hay base de datos que contenerizar. No hay Redis. El servidor es un proceso Go con go-libp2p — nada más. La imagen final pesa ~15MB. El `docker-compose.yml` solo existe para desarrollo local.

### ADR-004: net/http de Stdlib para /metrics y /healthz, Sin Framework Web

**Decisión:** Los endpoints HTTP del servidor usan `net/http` de la stdlib de Go, no Gin ni ningún framework.  
**Justificación:** Los únicos endpoints HTTP son `/metrics` (Prometheus handler) y `/healthz`/`/readyz`. Estos son tan simples que añadir un framework como Gin sería over-engineering. La stdlib de Go es perfectamente adecuada para dos handlers estáticos.

### ADR-005: Sin gRPC — Los Peers Hablan libp2p

**Decisión:** Este servidor no expone endpoints gRPC.  
**Justificación:** Los nodos ERP-CORE se conectan a este servidor usando el protocolo nativo de go-libp2p/rust-libp2p. No hay API de alto nivel que exponer — el servidor responde a los protocolos libp2p automáticamente (DHT queries, relay reservations, AutoNAT probes, identify handshakes). Un servidor gRPC sería una capa innecesaria encima de go-libp2p.

### ADR-006: t4g.nano como Instancia de Producción

**Decisión:** Usar instancia `t4g.nano` (2 vCPU, 0.5GB RAM, ~$3.50/mes) para producción inicial.  
**Justificación:** El binario Go + go-libp2p usa ~30–50MB de RAM. La tabla DHT con 1000 peers usa ~5MB. Los slots de relay (máx 128) son estructuras en memoria triviales. La instancia tiene excelente throughput de red para su precio. Si el CPU supera 70% sostenido, se puede subir a t4g.micro (~$7.50/mes) sin cambios en Terraform más allá de `instance_type`.

---

*Este documento requiere aprobación antes de comenzar el desarrollo del roadmap.*  
*Los cambios a este documento después de la aprobación requieren una entrada ADR explicando la modificación.*
