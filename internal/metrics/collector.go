package metrics

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	dhtpkg "github.com/bytsers/erp-p2p-cloud/internal/dht"
)

// Collector holds custom Prometheus metrics for the P2P node.
type Collector struct {
	startTime time.Time

	// Gauges (updated via polling)
	peersConnected     prometheus.Gauge
	dhtWANRoutingTable prometheus.Gauge
	dhtLANRoutingTable prometheus.Gauge
	uptimeSeconds      prometheus.Gauge

	// Counters (updated via event bus)
	peersConnectedTotal    prometheus.Counter
	peersDisconnectedTotal prometheus.Counter
	reachabilityChanges    *prometheus.CounterVec
}

// NewCollector creates and registers custom Prometheus metrics.
func NewCollector(reg prometheus.Registerer) *Collector {
	c := &Collector{
		startTime: time.Now(),
		peersConnected: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "erp_p2p_peers_connected",
			Help: "Current number of connected peers.",
		}),
		dhtWANRoutingTable: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "erp_p2p_dht_wan_routing_table_size",
			Help: "Number of peers in the WAN DHT routing table.",
		}),
		dhtLANRoutingTable: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "erp_p2p_dht_lan_routing_table_size",
			Help: "Number of peers in the LAN DHT routing table.",
		}),
		uptimeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "erp_p2p_uptime_seconds",
			Help: "Time in seconds since the node started.",
		}),
		peersConnectedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "erp_p2p_peers_connected_total",
			Help: "Total number of peer connection events.",
		}),
		peersDisconnectedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "erp_p2p_peers_disconnected_total",
			Help: "Total number of peer disconnection events.",
		}),
		reachabilityChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "erp_p2p_reachability_changes_total",
			Help: "Total number of reachability change events by type.",
		}, []string{"reachability"}),
	}

	reg.MustRegister(
		c.peersConnected,
		c.dhtWANRoutingTable,
		c.dhtLANRoutingTable,
		c.uptimeSeconds,
		c.peersConnectedTotal,
		c.peersDisconnectedTotal,
		c.reachabilityChanges,
	)

	return c
}

// StartCollection launches goroutines that update metrics via polling and event bus.
// It blocks until ctx is cancelled.
func (c *Collector) StartCollection(ctx context.Context, h host.Host, d *dhtpkg.DualDHT, logger *zerolog.Logger) {
	go c.pollMetrics(ctx, h, d, logger)
	go c.subscribeConnectedness(ctx, h, logger)
	go c.subscribeReachability(ctx, h, logger)
}

func (c *Collector) pollMetrics(ctx context.Context, h host.Host, d *dhtpkg.DualDHT, logger *zerolog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	update := func() {
		c.uptimeSeconds.Set(time.Since(c.startTime).Seconds())
		c.peersConnected.Set(float64(len(h.Network().Peers())))
		inner := d.Inner()
		c.dhtWANRoutingTable.Set(float64(inner.WAN.RoutingTable().Size()))
		c.dhtLANRoutingTable.Set(float64(inner.LAN.RoutingTable().Size()))
	}

	// Initial update
	update()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update()
		}
	}
}

func (c *Collector) subscribeConnectedness(ctx context.Context, h host.Host, logger *zerolog.Logger) {
	sub, err := h.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		logger.Warn().Err(err).Msg("failed to subscribe to peer connectedness events")
		return
	}
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-sub.Out():
			if !ok {
				return
			}
			evt := e.(event.EvtPeerConnectednessChanged)
			switch evt.Connectedness {
			case network.Connected:
				c.peersConnectedTotal.Inc()
			case network.NotConnected:
				c.peersDisconnectedTotal.Inc()
			}
		}
	}
}

func (c *Collector) subscribeReachability(ctx context.Context, h host.Host, logger *zerolog.Logger) {
	sub, err := h.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		logger.Warn().Err(err).Msg("failed to subscribe to reachability events")
		return
	}
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-sub.Out():
			if !ok {
				return
			}
			evt := e.(event.EvtLocalReachabilityChanged)
			c.reachabilityChanges.WithLabelValues(evt.Reachability.String()).Inc()
			logger.Info().
				Str("reachability", evt.Reachability.String()).
				Msg("local reachability changed")
		}
	}
}
