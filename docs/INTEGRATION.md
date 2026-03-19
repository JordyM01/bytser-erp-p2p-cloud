# Guia de Integracion ERP-CORE con ERP-P2P-CLOUD

**Audiencia:** Equipo de desarrollo ERP-CORE (Rust / rust-libp2p)
**Version:** 1.0 — Marzo 2026

---

## Indice

1. [Resumen](#1-resumen)
2. [Que hace el servidor P2P por tu nodo](#2-que-hace-el-servidor-p2p-por-tu-nodo)
3. [Prerequisitos](#3-prerequisitos)
4. [Configuracion del Bootstrap Peer](#4-configuracion-del-bootstrap-peer)
5. [Escalera de Conexion ICE](#5-escalera-de-conexion-ice)
6. [Implementacion en rust-libp2p](#6-implementacion-en-rust-libp2p)
7. [Descubrimiento de Peers (DHT)](#7-descubrimiento-de-peers-dht)
8. [NAT Traversal: Hole Punching y Relay](#8-nat-traversal-hole-punching-y-relay)
9. [Manejo de Reconexion](#9-manejo-de-reconexion)
10. [Verificacion de Integracion](#10-verificacion-de-integracion)
11. [Limites del Relay](#11-limites-del-relay)
12. [Troubleshooting](#12-troubleshooting)
13. [Referencia de Protocolos](#13-referencia-de-protocolos)

---

## 1. Resumen

ERP-P2P-CLOUD es un servidor relay stateless que permite a los nodos ERP-CORE descubrirse y conectarse entre si, incluso detras de NAT/firewall. El servidor:

- **No conoce** el contenido de los paquetes (relay ciego, todo va cifrado extremo a extremo)
- **No almacena** datos en disco (todo en RAM)
- **No autentica** usuarios ni organizaciones (eso es responsabilidad de ERP-CLOUD-SYNC)

Tu nodo ERP-CORE se conecta al servidor usando protocolos nativos de libp2p. No hay API REST, no hay gRPC, no hay WebSocket custom. Todo es libp2p puro.

---

## 2. Que hace el servidor P2P por tu nodo

| Servicio | Que hace | Protocolo libp2p |
|---|---|---|
| **Bootstrap DHT** | Punto de entrada a la red Kademlia para descubrir otros peers | `/ipfs/kad/1.0.0` |
| **AutoNAT** | Le dice a tu nodo su IP publica ("tu IP es X.X.X.X") | `/libp2p/autonat/2.0.0` |
| **Hole Punching (DCUtR)** | Coordina perforacion simultanea de NAT para conexion directa | `/libp2p/dcutr` |
| **Circuit Relay v2** | Triangula trafico cuando hole-punch falla (~10-20% de casos) | `/libp2p/circuit/relay/0.2.0/hop` |
| **Identify** | Intercambia metadata (PeerID, protocolos, agent version) | `/ipfs/id/1.0.0` |

---

## 3. Prerequisitos

### Dependencias Rust (Cargo.toml)

```toml
[dependencies]
libp2p = { version = "0.54", features = [
    "tokio",
    "noise",
    "yamux",
    "tcp",
    "quic",
    "dns",
    "kad",
    "dcutr",
    "relay",
    "autonat",
    "identify",
    "ping",
    "mdns",
] }
tokio = { version = "1", features = ["full"] }
```

> **Importante:** rust-libp2p y go-libp2p son interoperables a nivel de protocolo. No necesitas nada especial para conectar con un servidor Go.

### Datos que necesitas del equipo P2P

| Dato | Ejemplo | Donde obtenerlo |
|---|---|---|
| PeerID del servidor | `12D3KooWxxxxxx...` | Equipo infra / Secrets Manager |
| Dominio | `relay.bytsers.com` | DNS Route53 |
| Puerto TCP | `4001` | Fijo |
| Puerto QUIC | `4001/udp` | Fijo |

---

## 4. Configuracion del Bootstrap Peer

El servidor P2P tiene un PeerID estable (derivado de un keypair Ed25519 almacenado en AWS Secrets Manager). Este PeerID **no cambia entre reinicios**.

### Multiaddrs del servidor

```
/dns4/relay.bytsers.com/tcp/4001/p2p/12D3KooW[PEER_ID]
/dns4/relay.bytsers.com/udp/4001/quic-v1/p2p/12D3KooW[PEER_ID]
```

> El PeerID real se proporcionara cuando el servidor este desplegado en produccion. Para desarrollo local, el servidor genera un PeerID temporal en `.local/identity.key`.

### Config en ERP-CORE

```toml
# erp-core config
[p2p]
bootstrap_peers = [
    "/dns4/relay.bytsers.com/tcp/4001/p2p/12D3KooWXXXXXX",
    "/dns4/relay.bytsers.com/udp/4001/quic-v1/p2p/12D3KooWXXXXXX",
]
```

---

## 5. Escalera de Conexion ICE

Cuando un nodo ERP-CORE arranca, ejecuta esta escalera en orden. Se detiene en el primer paso exitoso:

```
Paso 1: mDNS (LAN)
   Tu nodo busca peers en la red local via multicast DNS.
   Si el otro nodo esta en la misma WiFi/Ethernet -> conexion directa.
   Costo: $0. Latencia: <5ms.
   El servidor P2P ni se entera.
        |
        | Si el peer NO esta en la misma red...
        v
Paso 2: DHT + Hole Punching (WAN directo)
   1. Tu nodo se conecta al servidor P2P (bootstrap peer)
   2. AutoNAT te informa tu IP publica
   3. Tu nodo publica su PeerID + multiaddrs en el DHT
   4. Descubres el PeerID del otro nodo en el DHT
   5. DCUtR coordina hole-punch UDP simultaneo
   6. Si exitoso -> conexion directa (el relay se desconecta)
   Tasa de exito: 80-90%. Latencia: 20-80ms.
        |
        | Si hole-punch falla (NAT simetrico, firewall estricto)...
        v
Paso 3: Circuit Relay v2 (fallback)
   Tu nodo reserva un slot en el relay del servidor.
   Trafico: NodoA -> relay.bytsers.com -> NodoB
   Limitado: 128KB/circuito, 5 min max.
   Solo para ~10-20% de nodos detras de NAT simetrico.
```

---

## 6. Implementacion en rust-libp2p

### 6.1 Crear el Swarm

```rust
use libp2p::{
    identity, noise, yamux, tcp, quic, dns,
    kad, dcutr, relay, autonat, identify, ping, mdns,
    swarm::NetworkBehaviour, Multiaddr, PeerId, SwarmBuilder,
};
use std::str::FromStr;

// Generar o cargar identidad persistente del nodo ERP-CORE
let local_key = identity::Keypair::generate_ed25519();
let local_peer_id = PeerId::from(local_key.public());

// PeerID del servidor relay (proporcionado por equipo infra)
let relay_peer_id = PeerId::from_str("12D3KooWXXXXXX").unwrap();
let relay_addr_tcp: Multiaddr =
    "/dns4/relay.bytsers.com/tcp/4001".parse().unwrap();
let relay_addr_quic: Multiaddr =
    "/dns4/relay.bytsers.com/udp/4001/quic-v1".parse().unwrap();

let mut swarm = SwarmBuilder::with_existing_identity(local_key)
    .with_tokio()
    .with_tcp(tcp::Config::default(), noise::Config::new, yamux::Config::default)?
    .with_quic()
    .with_dns()?
    .with_relay_client(noise::Config::new, yamux::Config::default)?
    .with_behaviour(|keypair, relay_client| {
        Ok(MyBehaviour {
            kademlia: kad::Behaviour::new(
                local_peer_id,
                kad::store::MemoryStore::new(local_peer_id),
            ),
            dcutr: dcutr::Behaviour::new(local_peer_id),
            relay_client,
            autonat: autonat::Behaviour::new(local_peer_id, Default::default()),
            identify: identify::Behaviour::new(identify::Config::new(
                "/bytser/erp-core/1.0.0".to_string(),
                keypair.public(),
            )),
            ping: ping::Behaviour::default(),
            mdns: mdns::tokio::Behaviour::new(
                mdns::Config::default(),
                local_peer_id,
            )?,
        })
    })?
    .build();
```

### 6.2 Definir el NetworkBehaviour

```rust
#[derive(NetworkBehaviour)]
struct MyBehaviour {
    kademlia: kad::Behaviour<kad::store::MemoryStore>,
    dcutr: dcutr::Behaviour,
    relay_client: relay::client::Behaviour,
    autonat: autonat::Behaviour,
    identify: identify::Behaviour,
    ping: ping::Behaviour,
    mdns: mdns::tokio::Behaviour,
}
```

### 6.3 Conectar al Bootstrap Peer

```rust
// Escuchar en puertos locales
swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;
swarm.listen_on("/ip4/0.0.0.0/udp/0/quic-v1".parse()?)?;

// Agregar direccion del relay al peerstore
swarm.behaviour_mut().kademlia.add_address(
    &relay_peer_id,
    relay_addr_tcp.clone(),
);
swarm.behaviour_mut().kademlia.add_address(
    &relay_peer_id,
    relay_addr_quic.clone(),
);

// Conectar al bootstrap peer
swarm.dial(
    relay_addr_tcp.with_p2p(relay_peer_id).unwrap()
)?;

// Iniciar bootstrap DHT
swarm.behaviour_mut().kademlia.bootstrap()?;
```

### 6.4 Escuchar via Relay (para recibir conexiones entrantes)

```rust
// Escuchar a traves del relay para recibir conexiones de otros nodos
// Esto reserva un slot en el Circuit Relay v2 del servidor
let relay_listen_addr = relay_addr_tcp
    .with(libp2p::multiaddr::Protocol::P2p(relay_peer_id))
    .with(libp2p::multiaddr::Protocol::P2pCircuit);

swarm.listen_on(relay_listen_addr)?;
```

---

## 7. Descubrimiento de Peers (DHT)

### Publicar tu nodo en el DHT

Una vez conectado al bootstrap peer, tu nodo se registra automaticamente en la tabla Kademlia. Otros nodos pueden encontrarte por tu PeerID.

### Descubrir nodos de la misma organizacion

El mecanismo recomendado para descubrir peers de la misma organizacion es **Kademlia Provider Records**:

```rust
// Publicar: "yo proveo datos para la organizacion X"
let org_key = kad::RecordKey::new(&format!("/bytser/org/{}", org_id));
swarm.behaviour_mut().kademlia.start_providing(org_key.clone())?;

// Buscar: "quien mas provee datos para la organizacion X?"
swarm.behaviour_mut().kademlia.get_providers(org_key);
```

Cuando recibas eventos `KademliaEvent::OutboundQueryProgressed` con `GetProvidersOk`, tendras los PeerIDs de los nodos de tu organizacion. Luego puedes conectarte directamente a ellos.

### Alternativa: Rendezvous (futuro)

Si en el futuro se necesita un mecanismo de descubrimiento mas fino (por ejemplo, descubrir peers por sucursal), se puede implementar el protocolo Rendezvous (`/rendezvous/1.0.0`) sobre el mismo servidor.

---

## 8. NAT Traversal: Hole Punching y Relay

### Hole Punching (DCUtR)

El proceso es **automatico** si tienes `dcutr::Behaviour` en tu swarm. Cuando dos nodos se descubren y ambos estan detras de NAT:

1. Ambos se conectan al relay
2. DCUtR coordina un intento simultaneo de conexion directa
3. Si exitoso, el trafico pasa directamente entre los nodos (sin relay)

No necesitas codigo adicional para esto. Solo asegurate de que:
- `dcutr::Behaviour` esta en tu `NetworkBehaviour`
- `relay::client::Behaviour` esta configurado
- Tu nodo escucha via el relay (ver seccion 6.4)

### Eventos relevantes

```rust
loop {
    match swarm.select_next_some().await {
        // Hole-punch exitoso: conexion directa establecida
        SwarmEvent::Behaviour(MyBehaviourEvent::Dcutr(
            dcutr::Event::DirectConnectionUpgraded { remote_peer_id, .. }
        )) => {
            println!("Conexion directa con {remote_peer_id} (hole-punch exitoso)");
        }

        // Nuevo peer descubierto via mDNS (LAN)
        SwarmEvent::Behaviour(MyBehaviourEvent::Mdns(
            mdns::Event::Discovered(peers)
        )) => {
            for (peer_id, addr) in peers {
                swarm.behaviour_mut().kademlia.add_address(&peer_id, addr);
            }
        }

        // AutoNAT: tu nodo ahora sabe su IP publica
        SwarmEvent::Behaviour(MyBehaviourEvent::Autonat(
            autonat::Event::StatusChanged { old, new }
        )) => {
            println!("Reachability cambio: {old:?} -> {new:?}");
        }

        // Nuevo peer conectado
        SwarmEvent::ConnectionEstablished { peer_id, .. } => {
            println!("Conectado a {peer_id}");
        }

        _ => {}
    }
}
```

---

## 9. Manejo de Reconexion

El servidor P2P es stateless. Si se reinicia, tu nodo debe reconectarse automaticamente. rust-libp2p maneja reconexion a nivel de transporte, pero debes implementar logica de retry para el bootstrap:

```rust
// Reintentar conexion al bootstrap peer cada 30 segundos si se pierde
let mut bootstrap_interval = tokio::time::interval(Duration::from_secs(30));

loop {
    tokio::select! {
        _ = bootstrap_interval.tick() => {
            if !swarm.is_connected(&relay_peer_id) {
                let _ = swarm.dial(
                    relay_addr_tcp.clone().with_p2p(relay_peer_id).unwrap()
                );
                let _ = swarm.behaviour_mut().kademlia.bootstrap();
            }
        }
        event = swarm.select_next_some() => {
            // ... manejar eventos
        }
    }
}
```

**Importante:** El PeerID del servidor es estable (nunca cambia entre reinicios). La unica situacion donde cambiaria es una rotacion de identidad, que se coordinaria con el equipo ERP-CORE con anticipacion.

---

## 10. Verificacion de Integracion

### 10.1 Contra servidor local (desarrollo)

```bash
# Terminal 1: levantar servidor P2P en modo dev
cd bytser-erp-p2p-cloud
make dev
# Anotar el PeerID que aparece en los logs:
# "identity loaded" peer_id=12D3KooW...
```

Luego en tu nodo ERP-CORE, configura el bootstrap peer con la direccion local:

```toml
[p2p]
bootstrap_peers = [
    "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...",
]
```

### 10.2 Checklist de verificacion

| Paso | Como verificar | Esperado |
|---|---|---|
| Conexion al bootstrap | Log `"Conectado a 12D3KooW..."` en ERP-CORE | Conexion TCP o QUIC exitosa |
| Identify | Log `"peer identified"` en el servidor P2P | Agent version intercambiado |
| AutoNAT | Evento `autonat::Event::StatusChanged` en ERP-CORE | Tu nodo conoce su IP publica |
| DHT Bootstrap | `kademlia.bootstrap()` retorna `Ok` | Tu nodo esta en la tabla DHT |
| Provider records | `start_providing()` y `get_providers()` funcionan | Nodos de la misma org se descubren |
| Hole punch | Log `"DirectConnectionUpgraded"` | Conexion directa (sin relay) |
| Relay fallback | Tu nodo puede recibir conexiones via relay | Circuit Relay v2 funcional |

### 10.3 Verificar desde metricas del servidor

```bash
# Peers conectados (debe incluir tu nodo)
curl -s localhost:9090/metrics | grep erp_p2p_peers_connected

# Conexiones totales
curl -s localhost:9090/metrics | grep erp_p2p_peers_connected_total

# DHT routing table (debe tener al menos 1 peer)
curl -s localhost:9090/metrics | grep erp_p2p_dht_wan_routing_table_size
```

---

## 11. Limites del Relay

El Circuit Relay v2 tiene limites estrictos para prevenir abuso. Tu nodo debe estar preparado para trabajar dentro de estos limites:

| Limite | Valor | Implicacion |
|---|---|---|
| Max reservaciones | 128 | Maximo 128 nodos pueden reservar slots simultaneamente |
| Max circuitos | 64 | Maximo 64 circuitos relay activos a la vez |
| Max por IP | 8 | Maximo 8 reservaciones desde la misma IP publica |
| TTL reservacion | 1 hora | Debes renovar la reservacion cada hora |
| Duracion circuito | 5 minutos | Cada circuito se cierra despues de 5 min |
| Datos por circuito | 128 KB | Suficiente para handshake, no para transferencia masiva |

**Consecuencia practica:** El relay es solo para el handshake inicial y bootstrapping de sincronizacion. Los datos de negocio pesados (changesets CRDT) **deben** fluir por conexion directa (hole-punch o LAN). Si tu nodo depende del relay para transferir datos, algo esta mal.

---

## 12. Troubleshooting

### "No puedo conectar al bootstrap peer"

1. Verificar que el servidor esta corriendo: `curl http://relay.bytsers.com:8080/healthz`
2. Verificar que el PeerID en tu config coincide con el del servidor
3. Verificar conectividad de red: `nc -zv relay.bytsers.com 4001`
4. En desarrollo local: verificar que `make dev` esta corriendo y usar `127.0.0.1`

### "AutoNAT dice que soy Private/Unknown"

Esto es normal si estas en una red con NAT simetrico o firewall estricto. Tu nodo usara el relay como fallback. No es un error.

### "DHT bootstrap no encuentra peers"

En desarrollo local sin otros nodos, el DHT estara vacio. Esto es esperado. Los peers aparecen cuando otros nodos ERP-CORE se conectan al mismo servidor.

### "Hole-punch falla siempre"

Algunas redes corporativas bloquean UDP completamente. En ese caso, tu nodo usara el Circuit Relay v2. Verifica:
- Que tu nodo escucha via el relay (seccion 6.4)
- Que `dcutr::Behaviour` esta en tu swarm
- Que el servidor reporta tu nodo como conectado (`erp_p2p_peers_connected`)

### "Circuito relay se cierra despues de 5 minutos"

Esto es por diseno. El relay es un puente temporal. Tu nodo debe:
1. Usar el relay para el handshake inicial
2. Intentar hole-punch para conexion directa
3. Si hole-punch falla, renovar el circuito relay

### "Conexion se pierde cuando el servidor reinicia"

Esto es normal (el servidor es stateless). Tu nodo debe reconectarse automaticamente (ver seccion 9). El DHT se reconstruye en segundos cuando los nodos se reconectan.

---

## 13. Referencia de Protocolos

Protocolos libp2p soportados por el servidor y que tu nodo debe implementar:

| Protocolo | ID | Requerido | Funcion |
|---|---|---|---|
| Kademlia DHT | `/ipfs/kad/1.0.0` | Si | Descubrimiento de peers |
| Circuit Relay v2 (client) | `/libp2p/circuit/relay/0.2.0/stop` | Si | Recibir conexiones via relay |
| DCUtR | `/libp2p/dcutr` | Si | Hole punching |
| AutoNAT | `/libp2p/autonat/2.0.0` | Si | Deteccion de NAT |
| Identify | `/ipfs/id/1.0.0` | Si | Intercambio de metadata |
| Ping | `/ipfs/ping/1.0.0` | Recomendado | Keep-alive |
| mDNS | (multicast local) | Recomendado | Descubrimiento LAN |
| Noise | Handshake de transporte | Si | Cifrado de conexiones |
| Yamux | Multiplexor de streams | Si | Multiplexacion sobre TCP |

### Transporte preferido

| Transporte | Multiaddr | Notas |
|---|---|---|
| **QUIC** (preferido) | `/udp/4001/quic-v1` | 0-RTT, TLS 1.3 nativo, mejor para hole-punch |
| TCP + Noise + Yamux | `/tcp/4001` | Fallback si QUIC esta bloqueado |

---

## Contacto

- **Repositorio servidor:** `bytsers/erp-p2p-cloud`
- **Health check:** `http://relay.bytsers.com:8080/healthz`
- **Metricas:** `http://relay.bytsers.com:9090/metrics` (solo red interna)
- **Dashboard Grafana:** `http://localhost:3000` (via Docker Compose en desarrollo)
