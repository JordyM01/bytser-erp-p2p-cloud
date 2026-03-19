package relay

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/rs/zerolog"
)

const hopProtocol = protocol.ID("/libp2p/circuit/relay/0.2.0/hop")

// Config holds Circuit Relay v2 resource limits.
type Config struct {
	MaxReservations       int
	MaxCircuits           int
	MaxReservationsPerIP  int
	MaxReservationsPerASN int
	BufferSize            int
	ReservationTTL        time.Duration
	LimitDuration         time.Duration
	LimitData             int64
}

// HostOptions returns the libp2p options to enable Circuit Relay v2 on a host.
// The relay is managed by the host's RelayManager and starts when reachability
// becomes public. Uses the built-in Prometheus MetricsTracer (namespace libp2p_relaysvc).
func HostOptions(cfg *Config) []relayv2.Option {
	return []relayv2.Option{
		relayv2.WithResources(relayv2.Resources{
			Limit: &relayv2.RelayLimit{
				Duration: cfg.LimitDuration,
				Data:     cfg.LimitData,
			},
			ReservationTTL:        cfg.ReservationTTL,
			MaxReservations:       cfg.MaxReservations,
			MaxCircuits:           cfg.MaxCircuits,
			MaxReservationsPerIP:  cfg.MaxReservationsPerIP,
			MaxReservationsPerASN: cfg.MaxReservationsPerASN,
			BufferSize:            cfg.BufferSize,
		}),
		relayv2.WithMetricsTracer(relayv2.NewMetricsTracer()),
	}
}

// ReadinessCheck returns a function that verifies the relay hop protocol
// is registered on the host (i.e., the relay service has started).
func ReadinessCheck(h host.Host, logger *zerolog.Logger) func() error {
	return func() error {
		for _, p := range h.Mux().Protocols() {
			if p == hopProtocol {
				return nil
			}
		}
		logger.Warn().Msg("relay hop protocol not yet registered")
		return fmt.Errorf("relay service not active")
	}
}
