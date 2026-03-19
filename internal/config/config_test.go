package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	require.NoError(t, err)
}

func TestLoadConfig_FromYAML(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")

	writeYAML(t, configDir, "config.test.yaml", `
app:
  env: test
  version: "1.2.3"
server:
  health_port: 9999
  metrics_port: 9998
logging:
  level: debug
  format: pretty
`)

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg, err := LoadConfig("test")
	require.NoError(t, err)

	assert.Equal(t, "test", cfg.App.Env)
	assert.Equal(t, "1.2.3", cfg.App.Version)
	assert.Equal(t, 9999, cfg.Server.HealthPort)
	assert.Equal(t, 9998, cfg.Server.MetricsPort)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "pretty", cfg.Logging.Format)
}

func TestLoadConfig_EnvVarOverride(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")

	writeYAML(t, configDir, "config.test.yaml", `
app:
  env: test
server:
  health_port: 8080
  metrics_port: 9090
logging:
  level: info
`)

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("P2P_LOGGING_LEVEL", "debug")

	cfg, err := LoadConfig("test")
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoadConfig_DefaultsApplied(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")

	writeYAML(t, configDir, "config.test.yaml", `
app:
  env: test
`)

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg, err := LoadConfig("test")
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.HealthPort)
	assert.Equal(t, 9090, cfg.Server.MetricsPort)
	assert.Equal(t, "/ip4/0.0.0.0/tcp/4001", cfg.P2P.ListenTCP)
	assert.Equal(t, "/ip4/0.0.0.0/udp/4001/quic-v1", cfg.P2P.ListenQUIC)
	assert.Equal(t, "/bytser/erp", cfg.P2P.DHTNamespace)
	assert.Equal(t, 128, cfg.P2P.RelayMaxReservations)
	assert.Equal(t, 64, cfg.P2P.RelayMaxCircuits)
	assert.Equal(t, time.Hour, cfg.P2P.RelayTTL)
	assert.Equal(t, 5*time.Minute, cfg.P2P.RelayMaxCircuitDur)
	assert.Equal(t, int64(131072), cfg.P2P.RelayMaxCircuitBytes)
	assert.True(t, cfg.P2P.AutoNATEnabled)
	assert.Equal(t, 30*time.Second, cfg.P2P.AutoNATThrottlePeer)
	assert.Equal(t, 900, cfg.P2P.ConnMgrLowWater)
	assert.Equal(t, 1000, cfg.P2P.ConnMgrHighWater)
	assert.Equal(t, 30*time.Second, cfg.P2P.ConnMgrGrace)
	assert.Equal(t, "us-east-1", cfg.AWS.Region)
}

func TestLoadConfig_ValidationError(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")

	writeYAML(t, configDir, "config.test.yaml", `
app:
  env: ""
server:
  health_port: 8080
  metrics_port: 9090
`)

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	_, err := LoadConfig("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.env is required")
}

func TestLoadConfig_NoConfigFile_UsesDefaults(t *testing.T) {
	tmp := t.TempDir()

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)

	assert.Equal(t, "dev", cfg.App.Env)
	assert.Equal(t, 8080, cfg.Server.HealthPort)
}
