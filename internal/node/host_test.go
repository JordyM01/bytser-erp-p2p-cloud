package node

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	relaypkg "github.com/bytsers/erp-p2p-cloud/internal/relay"
)

func buildTestHost(t *testing.T) host.Host {
	t.Helper()
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	id, err := newIdentityFromPrivKey(privKey)
	require.NoError(t, err)

	logger := zerolog.Nop()
	h, err := BuildHost(context.Background(), &HostConfig{
		Identity:      id,
		ListenTCP:     "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:    "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:    10,
		ConnMgrHigh:   20,
		ConnMgrGrace:  time.Second,
		UserAgent:     "test-agent/0.0.1",
		PrometheusReg: prometheus.NewRegistry(),
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func TestBuildHost_StartsAndListens(t *testing.T) {
	h := buildTestHost(t)
	assert.NotEmpty(t, h.ID())
	assert.Greater(t, len(h.Addrs()), 0, "host should have listen addresses")
}

func TestIntegration_TwoNodes_CanConnect(t *testing.T) {
	hA := buildTestHost(t)
	hB := buildTestHost(t)

	err := hA.Connect(context.Background(), hB.Peerstore().PeerInfo(hB.ID()))
	require.NoError(t, err)

	assert.Equal(t, network.Connected, hA.Network().Connectedness(hB.ID()))
	assert.Equal(t, network.Connected, hB.Network().Connectedness(hA.ID()))
}

func TestIntegration_Ping_RTT(t *testing.T) {
	hA := buildTestHost(t)
	hB := buildTestHost(t)

	err := hA.Connect(context.Background(), hB.Peerstore().PeerInfo(hB.ID()))
	require.NoError(t, err)

	rtt, err := PingPeer(context.Background(), hA, hB.ID())
	require.NoError(t, err)
	assert.Less(t, rtt, 100*time.Millisecond, "loopback ping should be fast")
}

func TestIntegration_Identify_UserAgent(t *testing.T) {
	hA := buildTestHost(t)
	hB := buildTestHost(t)

	// Subscribe to identify events on hA before connecting
	sub, err := hA.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	require.NoError(t, err)
	defer sub.Close()

	err = hA.Connect(context.Background(), hB.Peerstore().PeerInfo(hB.ID()))
	require.NoError(t, err)

	// Wait for identify to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatal("timed out waiting for identify event")
	case e := <-sub.Out():
		evt := e.(event.EvtPeerIdentificationCompleted)
		assert.Equal(t, hB.ID(), evt.Peer)
	}

	// Verify agent version in peerstore
	av, err := hA.Peerstore().Get(hB.ID(), "AgentVersion")
	require.NoError(t, err)
	assert.Equal(t, "test-agent/0.0.1", av)
}

func TestHostListeningCheck_Succeeds(t *testing.T) {
	h := buildTestHost(t)
	check := HostListeningCheck(h)
	assert.NoError(t, check())
}

func TestHostListeningCheck_FailsAfterClose(t *testing.T) {
	h := buildTestHost(t)
	check := HostListeningCheck(h)

	require.NoError(t, h.Close())
	err := check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no listen addresses")
}

func TestBuildHost_WithRelayAndAutoNAT(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	id, err := newIdentityFromPrivKey(privKey)
	require.NoError(t, err)

	logger := zerolog.Nop()
	h, err := BuildHost(context.Background(), &HostConfig{
		Identity:      id,
		ListenTCP:     "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:    "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:    10,
		ConnMgrHigh:   20,
		ConnMgrGrace:  time.Second,
		UserAgent:     "test-relay/0.0.1",
		PrometheusReg: prometheus.NewRegistry(),
		RelayConfig: &relaypkg.Config{
			MaxReservations:       128,
			MaxCircuits:           16,
			MaxReservationsPerIP:  8,
			MaxReservationsPerASN: 32,
			BufferSize:            2048,
			ReservationTTL:        time.Hour,
			LimitDuration:         2 * time.Minute,
			LimitData:             131072,
		},
		EnableNATService:    true,
		AutoNATGlobalLimit:  30,
		AutoNATPerPeerLimit: 3,
		AutoNATInterval:     time.Minute,
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })

	// ForceReachabilityPublic triggers relay service to start asynchronously.
	// Poll briefly for the relay hop protocol to be registered.
	require.Eventually(t, func() bool {
		for _, p := range h.Mux().Protocols() {
			if p == "/libp2p/circuit/relay/0.2.0/hop" {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "relay hop protocol should be registered")
}

func TestBuildHost_WithAutoNAT(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	id, err := newIdentityFromPrivKey(privKey)
	require.NoError(t, err)

	logger := zerolog.Nop()
	h, err := BuildHost(context.Background(), &HostConfig{
		Identity:            id,
		ListenTCP:           "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:          "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:          10,
		ConnMgrHigh:         20,
		ConnMgrGrace:        time.Second,
		UserAgent:           "test-autonat/0.0.1",
		PrometheusReg:       prometheus.NewRegistry(),
		EnableNATService:    true,
		AutoNATGlobalLimit:  30,
		AutoNATPerPeerLimit: 3,
		AutoNATInterval:     time.Minute,
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })

	assert.NotEmpty(t, h.ID())
	assert.Greater(t, len(h.Addrs()), 0)
}
