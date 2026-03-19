package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dhtpkg "github.com/bytsers/erp-p2p-cloud/internal/dht"
)

func TestNewCollector_RegistersMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	_ = NewCollector(reg)

	families, err := reg.Gather()
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}

	expected := []string{
		"erp_p2p_peers_connected",
		"erp_p2p_dht_wan_routing_table_size",
		"erp_p2p_dht_lan_routing_table_size",
		"erp_p2p_uptime_seconds",
		"erp_p2p_peers_connected_total",
		"erp_p2p_peers_disconnected_total",
	}
	for _, name := range expected {
		assert.True(t, names[name], "metric %q should be registered", name)
	}
}

func TestCollector_UptimeIncreases(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := NewCollector(reg)

	time.Sleep(10 * time.Millisecond)

	// Manually trigger uptime update (same logic as pollMetrics)
	c.uptimeSeconds.Set(time.Since(c.startTime).Seconds())

	val := testutil.ToFloat64(c.uptimeSeconds)
	assert.Greater(t, val, 0.0, "uptime should be > 0")
}

func TestCollector_PollingUpdatesPeersGauge(t *testing.T) {
	reg := prometheus.NewRegistry()

	hA, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.PrometheusRegisterer(reg))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hA.Close() })

	// Separate registry for hB to avoid duplicate registration
	hB, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.PrometheusRegisterer(prometheus.NewRegistry()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hB.Close() })

	// Connect hA → hB
	err = hA.Connect(context.Background(), peer.AddrInfo{ID: hB.ID(), Addrs: hB.Addrs()})
	require.NoError(t, err)

	// Create DHT on hA (no bootstrap)
	logger := zerolog.Nop()
	d, err := dhtpkg.NewDualDHT(context.Background(), hA, &dhtpkg.Config{}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	c := NewCollector(prometheus.NewRegistry())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.StartCollection(ctx, hA, d, &logger)

	// Wait for initial polling update
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(c.peersConnected) >= 1
	}, 5*time.Second, 50*time.Millisecond, "peers_connected should be >= 1")
}

func TestCollector_EventBusPeerCounters(t *testing.T) {
	reg := prometheus.NewRegistry()

	hA, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.PrometheusRegisterer(reg))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hA.Close() })

	hB, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.PrometheusRegisterer(prometheus.NewRegistry()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hB.Close() })

	// Create DHT on hA
	logger := zerolog.Nop()
	d, err := dhtpkg.NewDualDHT(context.Background(), hA, &dhtpkg.Config{}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	c := NewCollector(prometheus.NewRegistry())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.StartCollection(ctx, hA, d, &logger)

	// Connect hA → hB (should trigger EvtPeerConnectednessChanged)
	err = hA.Connect(context.Background(), peer.AddrInfo{ID: hB.ID(), Addrs: hB.Addrs()})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(c.peersConnectedTotal) >= 1
	}, 5*time.Second, 50*time.Millisecond, "peers_connected_total should be >= 1")
}
