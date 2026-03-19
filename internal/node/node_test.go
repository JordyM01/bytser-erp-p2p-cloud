package node

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dhtpkg "github.com/bytsers/erp-p2p-cloud/internal/dht"
	relaypkg "github.com/bytsers/erp-p2p-cloud/internal/relay"
)

func newTestIdentity(t *testing.T) *Identity {
	t.Helper()
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)
	id, err := newIdentityFromPrivKey(privKey)
	require.NoError(t, err)
	return id
}

func newTestNode(t *testing.T, bootstrapPeers []string) *Node {
	t.Helper()
	logger := zerolog.Nop()
	n, err := NewNode(context.Background(), &HostConfig{
		Identity:      newTestIdentity(t),
		ListenTCP:     "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:    "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:    10,
		ConnMgrHigh:   20,
		ConnMgrGrace:  time.Second,
		UserAgent:     "test-node/0.0.1",
		PrometheusReg: prometheus.NewRegistry(),
	}, &dhtpkg.Config{
		BootstrapPeers: bootstrapPeers,
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = n.Close() })
	return n
}

func TestNewNode_StartsSuccessfully(t *testing.T) {
	n := newTestNode(t, nil)

	assert.NotNil(t, n.Host())
	assert.NotNil(t, n.DHT())
	assert.NotEmpty(t, n.Host().ID())
}

func TestNewNode_WithRelayAndAutoNAT(t *testing.T) {
	logger := zerolog.Nop()
	n, err := NewNode(context.Background(), &HostConfig{
		Identity:      newTestIdentity(t),
		ListenTCP:     "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:    "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:    10,
		ConnMgrHigh:   20,
		ConnMgrGrace:  time.Second,
		UserAgent:     "test-node-relay/0.0.1",
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
	}, &dhtpkg.Config{
		BootstrapPeers: nil,
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = n.Close() })

	assert.NotNil(t, n.Host())
	assert.NotNil(t, n.DHT())
}

func TestNewNode_Close_Graceful(t *testing.T) {
	logger := zerolog.Nop()
	n, err := NewNode(context.Background(), &HostConfig{
		Identity:      newTestIdentity(t),
		ListenTCP:     "/ip4/127.0.0.1/tcp/0",
		ListenQUIC:    "/ip4/127.0.0.1/udp/0/quic-v1",
		ConnMgrLow:    10,
		ConnMgrHigh:   20,
		ConnMgrGrace:  time.Second,
		UserAgent:     "test-node-close/0.0.1",
		PrometheusReg: prometheus.NewRegistry(),
	}, &dhtpkg.Config{
		BootstrapPeers: nil,
	}, &logger)
	require.NoError(t, err)

	err = n.Close()
	assert.NoError(t, err)

	// Host should have no addresses after close
	assert.Empty(t, n.Host().Addrs())
}

func TestNewNode_ReadinessChecks(t *testing.T) {
	n := newTestNode(t, nil)

	hostCheck := HostListeningCheck(n.Host())
	assert.NoError(t, hostCheck())

	dhtCheck := n.DHT().ReadinessCheck()
	assert.NoError(t, dhtCheck())
}

func TestIntegration_TwoNodes_ConnectViaDHT(t *testing.T) {
	nA := newTestNode(t, nil)

	// Bootstrap node B from node A
	addrA := nA.Host().Addrs()[0]
	bootstrapAddr := fmt.Sprintf("%s/p2p/%s", addrA, nA.Host().ID())

	nB := newTestNode(t, []string{bootstrapAddr})

	// Verify DHT readiness on both nodes
	assert.NoError(t, nA.DHT().ReadinessCheck()())
	assert.NoError(t, nB.DHT().ReadinessCheck()())
}
