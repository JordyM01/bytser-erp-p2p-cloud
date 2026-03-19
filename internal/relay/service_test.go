package relay

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRelayConfig() *Config {
	return &Config{
		MaxReservations:       128,
		MaxCircuits:           16,
		MaxReservationsPerIP:  8,
		MaxReservationsPerASN: 32,
		BufferSize:            2048,
		ReservationTTL:        time.Hour,
		LimitDuration:         2 * time.Minute,
		LimitData:             131072,
	}
}

// buildRelayServer creates a host with relay service enabled via EnableRelayService.
func buildRelayServer(t *testing.T) host.Host {
	t.Helper()
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	cfg := testRelayConfig()
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.DisableRelay(),
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableRelayService(HostOptions(cfg)...),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func TestHostOptions_ReturnsNonEmpty(t *testing.T) {
	opts := HostOptions(testRelayConfig())
	assert.NotEmpty(t, opts)
}

func TestRelayServer_HopProtocolRegistered(t *testing.T) {
	h := buildRelayServer(t)

	// ForceReachabilityPublic triggers relay to start asynchronously
	require.Eventually(t, func() bool {
		for _, p := range h.Mux().Protocols() {
			if p == protocol.ID("/libp2p/circuit/relay/0.2.0/hop") {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "relay hop protocol should be registered")
}

func TestReadinessCheck_Active(t *testing.T) {
	h := buildRelayServer(t)
	logger := zerolog.Nop()

	// Wait for relay to start
	require.Eventually(t, func() bool {
		return ReadinessCheck(h, &logger)() == nil
	}, 5*time.Second, 50*time.Millisecond)
}

func TestReadinessCheck_Inactive(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	// Host without relay service
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.DisableRelay(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })

	logger := zerolog.Nop()
	check := ReadinessCheck(h, &logger)
	assert.Error(t, check())
}

func TestIntegration_TwoPeersConnectViaRelay(t *testing.T) {
	cfg := testRelayConfig()

	// 1. Create relay server with our config
	relayHost, err := libp2p.New(
		libp2p.EnableRelayService(HostOptions(cfg)...),
		libp2p.ForceReachabilityPublic(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = relayHost.Close() })

	relayInfo := peer.AddrInfo{ID: relayHost.ID(), Addrs: relayHost.Addrs()}

	// 2. Create peer behind relay (autorelay auto-reserves)
	peerBehind, err := libp2p.New(
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{relayInfo}),
		libp2p.ForceReachabilityPrivate(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = peerBehind.Close() })

	// 3. Create dialer
	dialer, err := libp2p.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = dialer.Close() })

	// 4. Set up peerstore for dialer to find relay + peer
	dialer.Peerstore().AddAddrs(relayHost.ID(), relayHost.Addrs(), time.Hour)
	dialer.Peerstore().AddAddr(peerBehind.ID(),
		ma.StringCast("/p2p/"+relayHost.ID().String()+"/p2p-circuit"),
		time.Hour,
	)

	// 5. Retry until relay reservation is established and connection works
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		return dialer.Connect(ctx, peer.AddrInfo{ID: peerBehind.ID()}) == nil
	}, 30*time.Second, 500*time.Millisecond, "should connect via relay")

	// 6. Verify connectivity (relay connections are "Limited" in libp2p v2)
	conn := dialer.Network().Connectedness(peerBehind.ID())
	assert.True(t, conn == network.Connected || conn == network.Limited,
		"expected Connected or Limited, got %s", conn)
}
