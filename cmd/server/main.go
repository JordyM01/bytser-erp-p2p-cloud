package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/bytsers/erp-p2p-cloud/internal/config"
	dhtpkg "github.com/bytsers/erp-p2p-cloud/internal/dht"
	"github.com/bytsers/erp-p2p-cloud/internal/health"
	"github.com/bytsers/erp-p2p-cloud/internal/metrics"
	"github.com/bytsers/erp-p2p-cloud/internal/node"
	relaypkg "github.com/bytsers/erp-p2p-cloud/internal/relay"
)

var version = "dev"

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	cfg, err := config.LoadConfig(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := initLogger(cfg.Logging)
	logger.Info().
		Str("version", version).
		Str("env", cfg.App.Env).
		Msg("ERP-P2P-CLOUD starting")

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Load identity
	identity, err := loadIdentity(appCtx, cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load identity")
	}
	logger.Info().
		Str("peer_id", identity.PeerID.String()).
		Msg("identity loaded")
	logger = logger.With().Str("peer_id", identity.PeerID.String()).Logger()

	// Build node (host + DHT + identify events)
	n, err := node.NewNode(appCtx, buildHostConfig(cfg, identity), &dhtpkg.Config{
		BootstrapPeers: cfg.P2P.BootstrapPeers,
	}, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create node")
	}

	// Start custom metrics collector
	collector := metrics.NewCollector(prometheus.DefaultRegisterer)
	collector.StartCollection(appCtx, n.Host(), n.DHT(), &logger)

	// HTTP health server with host + DHT + relay readiness checks
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.HandleHealthz)
	mux.Handle("/readyz", health.HandleReadyz(
		node.HostListeningCheck(n.Host()),
		n.DHT().ReadinessCheck(),
		relaypkg.ReadinessCheck(n.Host(), &logger),
	))

	healthAddr := fmt.Sprintf(":%d", cfg.Server.HealthPort)
	healthSrv := &http.Server{
		Addr:         healthAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", healthAddr).Msg("health server listening")
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("health server failed")
		}
	}()

	// Metrics server (Prometheus)
	var metricsSrv *http.Server
	if cfg.Metrics.Enabled {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", metrics.Handler())

		metricsAddr := fmt.Sprintf(":%d", cfg.Server.MetricsPort)
		metricsSrv = &http.Server{
			Addr:         metricsAddr,
			Handler:      metricsMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		go func() {
			logger.Info().Str("addr", metricsAddr).Msg("metrics server listening")
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal().Err(err).Msg("metrics server failed")
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigCh
	logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")

	// Cancel app context first (stops identify event listener)
	appCancel()

	// Close node (DHT → host)
	if err := n.Close(); err != nil {
		logger.Error().Err(err).Msg("node close error")
	}

	// Shutdown HTTP servers
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if metricsSrv != nil {
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("metrics server shutdown error")
		}
	}

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("health server shutdown error")
	}

	logger.Info().Msg("shutdown completed")
}

func buildHostConfig(cfg *config.Config, identity *node.Identity) *node.HostConfig {
	hc := &node.HostConfig{
		Identity:      identity,
		ListenTCP:     cfg.P2P.ListenTCP,
		ListenQUIC:    cfg.P2P.ListenQUIC,
		ConnMgrLow:    cfg.P2P.ConnMgrLowWater,
		ConnMgrHigh:   cfg.P2P.ConnMgrHighWater,
		ConnMgrGrace:  cfg.P2P.ConnMgrGrace,
		UserAgent:     "bytsers-erp-p2p-cloud/" + version,
		PrometheusReg: prometheus.DefaultRegisterer,
		RelayConfig: &relaypkg.Config{
			MaxReservations:       cfg.P2P.RelayMaxReservations,
			MaxCircuits:           cfg.P2P.RelayMaxCircuits,
			MaxReservationsPerIP:  8,
			MaxReservationsPerASN: 32,
			BufferSize:            2048,
			ReservationTTL:        cfg.P2P.RelayTTL,
			LimitDuration:         cfg.P2P.RelayMaxCircuitDur,
			LimitData:             cfg.P2P.RelayMaxCircuitBytes,
		},
	}

	if cfg.P2P.AutoNATEnabled {
		hc.EnableNATService = true
		hc.AutoNATGlobalLimit = 30
		hc.AutoNATPerPeerLimit = 3
		hc.AutoNATInterval = cfg.P2P.AutoNATThrottlePeer
	}

	return hc
}

func loadIdentity(ctx context.Context, cfg *config.Config, logger *zerolog.Logger) (*node.Identity, error) {
	if cfg.App.Env == "dev" {
		logger.Info().Msg("loading local dev identity")
		return node.LoadOrGenerateLocal(".local/identity.key")
	}

	if cfg.P2P.IdentitySecretName == "" {
		return nil, fmt.Errorf("p2p.identity_secret_name is required for env %q", cfg.App.Env)
	}

	logger.Info().
		Str("secret_name", cfg.P2P.IdentitySecretName).
		Msg("loading identity from AWS Secrets Manager")

	var opts []func(*awsconfig.LoadOptions) error
	if cfg.AWS.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.AWS.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	var smOpts []func(*secretsmanager.Options)
	if cfg.AWS.SecretsEndpoint != "" {
		smOpts = append(smOpts, func(o *secretsmanager.Options) {
			o.BaseEndpoint = &cfg.AWS.SecretsEndpoint
		})
	}
	client := secretsmanager.NewFromConfig(awsCfg, smOpts...)

	return node.LoadFromSecretsManager(ctx, cfg.P2P.IdentitySecretName, client)
}

func initLogger(cfg config.LoggingConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var logger zerolog.Logger
	if cfg.Format == "pretty" {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().Timestamp().
			Str("service", "bytsers-erp-p2p-cloud").
			Logger()
	} else {
		logger = zerolog.New(os.Stdout).
			With().Timestamp().
			Str("service", "bytsers-erp-p2p-cloud").
			Logger()
	}

	return logger
}
