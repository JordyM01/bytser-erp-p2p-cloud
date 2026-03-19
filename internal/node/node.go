package node

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog"

	dhtpkg "github.com/bytsers/erp-p2p-cloud/internal/dht"
)

// Node orchestrates the libp2p host, DHT, and related services.
type Node struct {
	host   host.Host
	dht    *dhtpkg.DualDHT
	logger *zerolog.Logger
}

// NewNode creates a libp2p host and bootstraps a Dual DHT on top of it.
func NewNode(ctx context.Context, hostCfg *HostConfig, dhtCfg *dhtpkg.Config, logger *zerolog.Logger) (*Node, error) {
	h, err := BuildHost(ctx, hostCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("building host: %w", err)
	}

	d, err := dhtpkg.NewDualDHT(ctx, h, dhtCfg, logger)
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("creating DHT: %w", err)
	}

	LogIdentifyEvents(ctx, h, logger)

	return &Node{
		host:   h,
		dht:    d,
		logger: logger,
	}, nil
}

// Host returns the underlying libp2p host.
func (n *Node) Host() host.Host {
	return n.host
}

// DHT returns the Dual DHT instance.
func (n *Node) DHT() *dhtpkg.DualDHT {
	return n.dht
}

// Close shuts down the node in reverse order: DHT then host.
func (n *Node) Close() error {
	var firstErr error

	if err := n.dht.Close(); err != nil {
		n.logger.Error().Err(err).Msg("DHT close error")
		firstErr = err
	}

	if err := n.host.Close(); err != nil {
		n.logger.Error().Err(err).Msg("host close error")
		if firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
