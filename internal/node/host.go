package node

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	relaypkg "github.com/bytsers/erp-p2p-cloud/internal/relay"
)

// HostConfig holds the configuration needed to build a libp2p host.
type HostConfig struct {
	Identity     *Identity
	ListenTCP    string
	ListenQUIC   string
	ConnMgrLow   int
	ConnMgrHigh  int
	ConnMgrGrace time.Duration
	UserAgent    string

	// Relay v2 service (nil = disabled)
	RelayConfig *relaypkg.Config

	// AutoNAT service
	EnableNATService    bool
	AutoNATGlobalLimit  int
	AutoNATPerPeerLimit int
	AutoNATInterval     time.Duration

	// Prometheus registerer for built-in libp2p metrics (nil = prometheus.DefaultRegisterer)
	PrometheusReg prometheus.Registerer
}

// BuildHost creates and starts a libp2p host with the given configuration.
func BuildHost(ctx context.Context, cfg *HostConfig, logger *zerolog.Logger) (host.Host, error) {
	cm, err := connmgr.NewConnManager(cfg.ConnMgrLow, cfg.ConnMgrHigh, connmgr.WithGracePeriod(cfg.ConnMgrGrace))
	if err != nil {
		return nil, fmt.Errorf("creating connection manager: %w", err)
	}

	reg := cfg.PrometheusReg
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	opts := []libp2p.Option{
		libp2p.Identity(cfg.Identity.PrivKey),
		libp2p.ListenAddrStrings(cfg.ListenTCP, cfg.ListenQUIC),
		libp2p.ConnectionManager(cm),
		libp2p.NATPortMap(),
		libp2p.UserAgent(cfg.UserAgent),
		libp2p.DisableRelay(),
		libp2p.PrometheusRegisterer(reg),
	}

	if cfg.RelayConfig != nil {
		opts = append(opts, libp2p.EnableRelayService(relaypkg.HostOptions(cfg.RelayConfig)...))
		logger.Info().Msg("circuit relay v2 service enabled")
	}

	if cfg.EnableNATService {
		opts = append(opts,
			libp2p.EnableNATService(),
			libp2p.AutoNATServiceRateLimit(cfg.AutoNATGlobalLimit, cfg.AutoNATPerPeerLimit, cfg.AutoNATInterval),
			libp2p.ForceReachabilityPublic(),
		)
		logger.Info().Msg("AutoNAT service enabled (forced public reachability)")
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating libp2p host: %w", err)
	}

	logger.Info().
		Str("peer_id", h.ID().String()).
		Strs("listen_addrs", addrsToStrings(h)).
		Str("user_agent", cfg.UserAgent).
		Msg("libp2p host started")

	return h, nil
}

func addrsToStrings(h host.Host) []string {
	addrs := h.Addrs()
	s := make([]string, len(addrs))
	for i, a := range addrs {
		s[i] = a.String()
	}
	return s
}

// HostListeningCheck returns a readiness check function that verifies
// the host is listening on at least one address.
func HostListeningCheck(h host.Host) func() error {
	return func() error {
		if len(h.Addrs()) == 0 {
			return fmt.Errorf("libp2p host has no listen addresses")
		}
		return nil
	}
}

// PingPeer sends a single ping to the target peer and returns the round-trip time.
func PingPeer(ctx context.Context, h host.Host, target peer.ID) (time.Duration, error) {
	ch := ping.Ping(ctx, h, target)
	res := <-ch
	if res.Error != nil {
		return 0, fmt.Errorf("ping %s: %w", target, res.Error)
	}
	return res.RTT, nil
}

// LogIdentifyEvents subscribes to peer identification events and logs them.
// It runs in a goroutine and stops when ctx is cancelled.
func LogIdentifyEvents(ctx context.Context, h host.Host, logger *zerolog.Logger) {
	sub, err := h.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		logger.Warn().Err(err).Msg("failed to subscribe to identify events")
		return
	}

	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-sub.Out():
				if !ok {
					return
				}
				evt := e.(event.EvtPeerIdentificationCompleted)
				agent := "unknown"
				if av, err := h.Peerstore().Get(evt.Peer, "AgentVersion"); err == nil {
					if s, ok := av.(string); ok {
						agent = s
					}
				}
				logger.Debug().
					Str("peer_id", evt.Peer.String()).
					Str("agent_version", agent).
					Msg("peer identified")
			}
		}
	}()
}
