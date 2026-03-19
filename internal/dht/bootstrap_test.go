package dht

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestHost(t *testing.T) host.Host {
	t.Helper()
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.DisableRelay(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func TestNewDualDHT_NoBootstrapPeers(t *testing.T) {
	h := buildTestHost(t)
	logger := zerolog.Nop()

	dd, err := NewDualDHT(context.Background(), h, &Config{
		BootstrapPeers: []string{},
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dd.Close() })

	assert.True(t, dd.bootstrapped)
	assert.NotNil(t, dd.Inner())
}

func TestNewDualDHT_ReadinessCheck_Succeeds(t *testing.T) {
	h := buildTestHost(t)
	logger := zerolog.Nop()

	dd, err := NewDualDHT(context.Background(), h, &Config{
		BootstrapPeers: []string{},
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dd.Close() })

	check := dd.ReadinessCheck()
	assert.NoError(t, check())
}

func TestNewDualDHT_Close(t *testing.T) {
	h := buildTestHost(t)
	logger := zerolog.Nop()

	dd, err := NewDualDHT(context.Background(), h, &Config{
		BootstrapPeers: []string{},
	}, &logger)
	require.NoError(t, err)

	err = dd.Close()
	assert.NoError(t, err)
}

func TestParseBootstrapPeers_Valid(t *testing.T) {
	// Use a well-known IPFS bootstrap peer multiaddr
	addr := "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
	peers, err := parseBootstrapPeers([]string{addr})
	require.NoError(t, err)
	require.Len(t, peers, 1)

	expectedID, err := peer.Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	require.NoError(t, err)
	assert.Equal(t, expectedID, peers[0].ID)
}

func TestParseBootstrapPeers_InvalidMultiaddr(t *testing.T) {
	_, err := parseBootstrapPeers([]string{"not-a-multiaddr"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid multiaddr")
}

func TestParseBootstrapPeers_MissingPeerID(t *testing.T) {
	_, err := parseBootstrapPeers([]string{"/ip4/104.131.131.82/tcp/4001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extracting peer info")
}

func TestIntegration_TwoNodes_DHTRouting(t *testing.T) {
	hA := buildTestHost(t)
	hB := buildTestHost(t)

	logger := zerolog.Nop()

	// Node A: DHT with no bootstrap peers
	ddA, err := NewDualDHT(context.Background(), hA, &Config{
		BootstrapPeers: []string{},
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ddA.Close() })

	// Node B: DHT bootstrapped from node A
	addrA := hA.Addrs()[0]
	bootstrapAddr := fmt.Sprintf("%s/p2p/%s", addrA, hA.ID())

	ddB, err := NewDualDHT(context.Background(), hB, &Config{
		BootstrapPeers: []string{bootstrapAddr},
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ddB.Close() })

	assert.True(t, ddA.bootstrapped)
	assert.True(t, ddB.bootstrapped)
}
