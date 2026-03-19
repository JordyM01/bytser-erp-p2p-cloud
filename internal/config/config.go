package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the ERP-P2P-CLOUD server.
type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Server  ServerConfig  `mapstructure:"server"`
	P2P     P2PConfig     `mapstructure:"p2p"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Logging LoggingConfig `mapstructure:"logging"`
	AWS     AWSConfig     `mapstructure:"aws"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	HealthPort  int `mapstructure:"health_port"`
	MetricsPort int `mapstructure:"metrics_port"`
}

// P2PConfig holds libp2p network settings.
type P2PConfig struct {
	ListenTCP            string        `mapstructure:"listen_tcp"`
	ListenQUIC           string        `mapstructure:"listen_quic"`
	ExternalIP           string        `mapstructure:"external_ip"`
	DHTNamespace         string        `mapstructure:"dht_namespace"`
	RelayMaxReservations int           `mapstructure:"relay_max_reservations"`
	RelayMaxCircuits     int           `mapstructure:"relay_max_circuits"`
	RelayTTL             time.Duration `mapstructure:"relay_ttl"`
	RelayMaxCircuitDur   time.Duration `mapstructure:"relay_max_circuit_dur"`
	RelayMaxCircuitBytes int64         `mapstructure:"relay_max_circuit_bytes"`
	AutoNATEnabled       bool          `mapstructure:"autonat_enabled"`
	AutoNATThrottlePeer  time.Duration `mapstructure:"autonat_throttle_peer"`
	ConnMgrLowWater      int           `mapstructure:"conn_mgr_low_water"`
	ConnMgrHighWater     int           `mapstructure:"conn_mgr_high_water"`
	ConnMgrGrace         time.Duration `mapstructure:"conn_mgr_grace"`
	IdentitySecretName   string        `mapstructure:"identity_secret_name"`
	BootstrapPeers       []string      `mapstructure:"bootstrap_peers"`
}

// MetricsConfig holds metrics-related settings.
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// AWSConfig holds AWS-related settings.
type AWSConfig struct {
	Region          string `mapstructure:"region"`
	SecretsEndpoint string `mapstructure:"secrets_endpoint"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.version", "0.0.0")

	v.SetDefault("server.health_port", 8080)
	v.SetDefault("server.metrics_port", 9090)

	v.SetDefault("p2p.listen_tcp", "/ip4/0.0.0.0/tcp/4001")
	v.SetDefault("p2p.listen_quic", "/ip4/0.0.0.0/udp/4001/quic-v1")
	v.SetDefault("p2p.dht_namespace", "/bytser/erp")
	v.SetDefault("p2p.relay_max_reservations", 128)
	v.SetDefault("p2p.relay_max_circuits", 64)
	v.SetDefault("p2p.relay_ttl", time.Hour)
	v.SetDefault("p2p.relay_max_circuit_dur", 5*time.Minute)
	v.SetDefault("p2p.relay_max_circuit_bytes", int64(131072))
	v.SetDefault("p2p.autonat_enabled", true)
	v.SetDefault("p2p.autonat_throttle_peer", 30*time.Second)
	v.SetDefault("p2p.bootstrap_peers", []string{})
	v.SetDefault("p2p.conn_mgr_low_water", 900)
	v.SetDefault("p2p.conn_mgr_high_water", 1000)
	v.SetDefault("p2p.conn_mgr_grace", 30*time.Second)

	v.SetDefault("metrics.enabled", true)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	v.SetDefault("aws.region", "us-east-1")
}

// LoadConfig reads configuration from config/config.{env}.yaml, applies
// defaults, and allows overrides via environment variables.
func LoadConfig(env string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName("config." + env)
	v.SetConfigType("yaml")
	v.AddConfigPath("config/")
	v.AddConfigPath(".")

	v.SetEnvPrefix("P2P")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// Config file not found is acceptable; we use defaults + env vars.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.App.Env == "" {
		return fmt.Errorf("config: app.env is required")
	}
	if cfg.Server.HealthPort <= 0 {
		return fmt.Errorf("config: server.health_port must be positive")
	}
	if cfg.Server.MetricsPort <= 0 {
		return fmt.Errorf("config: server.metrics_port must be positive")
	}
	return nil
}
