package dht

import (
	"context"
	"fmt"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
)

// Config holds the configuration for the Dual DHT.
type Config struct {
	BootstrapPeers []string // multiaddr strings with /p2p/ suffix
}

// DualDHT wraps a dual.DHT (WAN+LAN) with lifecycle management.
type DualDHT struct {
	dht          *dual.DHT
	bootstrapped bool
	logger       *zerolog.Logger
}

// NewDualDHT creates and bootstraps a Dual DHT (WAN+LAN) on the given host.
func NewDualDHT(ctx context.Context, h host.Host, cfg *Config, logger *zerolog.Logger) (*DualDHT, error) {
	peers, err := parseBootstrapPeers(cfg.BootstrapPeers)
	if err != nil {
		return nil, fmt.Errorf("parsing bootstrap peers: %w", err)
	}

	d, err := dual.New(ctx, h,
		dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.BootstrapPeers(peers...),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating dual DHT: %w", err)
	}

	if err := d.Bootstrap(ctx); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("bootstrapping DHT: %w", err)
	}

	logger.Info().
		Int("bootstrap_peers", len(peers)).
		Msg("DHT bootstrapped")

	return &DualDHT{
		dht:          d,
		bootstrapped: true,
		logger:       logger,
	}, nil
}

// Close shuts down the Dual DHT.
func (dd *DualDHT) Close() error {
	return dd.dht.Close()
}

// Inner returns the underlying dual.DHT for advanced use.
func (dd *DualDHT) Inner() *dual.DHT {
	return dd.dht
}

// ReadinessCheck returns a function that verifies the DHT has been bootstrapped.
func (dd *DualDHT) ReadinessCheck() func() error {
	return func() error {
		if !dd.bootstrapped {
			return fmt.Errorf("DHT not bootstrapped")
		}
		return nil
	}
}

// parseBootstrapPeers converts multiaddr strings to peer.AddrInfo.
// Each address must include a /p2p/ component with the peer ID.
func parseBootstrapPeers(addrs []string) ([]peer.AddrInfo, error) {
	peers := make([]peer.AddrInfo, 0, len(addrs))
	for _, s := range addrs {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			return nil, fmt.Errorf("invalid multiaddr %q: %w", s, err)
		}
		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, fmt.Errorf("extracting peer info from %q: %w", s, err)
		}
		peers = append(peers, *ai)
	}
	return peers, nil
}
