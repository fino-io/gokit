package accesslog

import (
	"time"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

const configPath = "transport.accessLog"

// Config controls access log filtering and sampling.
type Config struct {
	SlowThreshold time.Duration `json:"slowThreshold"`
	SampleEvery   uint64        `json:"sampleEvery"`
	HTTP          HTTPConfig    `json:"http"`
	GRPC          GRPCConfig    `json:"grpc"`
}

// HTTPConfig contains HTTP-specific filters.
type HTTPConfig struct {
	SkipPaths []string `json:"skipPaths"`
}

// GRPCConfig contains gRPC-specific filters.
type GRPCConfig struct {
	SkipMethods []string `json:"skipMethods"`
}

// DefaultConfig returns production-safe access log defaults.
func DefaultConfig() Config {
	return Config{
		SlowThreshold: 500 * time.Millisecond,
		SampleEvery:   100,
		HTTP: HTTPConfig{SkipPaths: []string{
			"/healthz",
			"/readyz",
			"/metrics",
		}},
		GRPC: GRPCConfig{SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		}},
	}
}

// LoadConfig reads transport.accessLog over the defaults.
func LoadConfig() Config {
	cfg := DefaultConfig()
	if err := config.ScanFrom(&cfg, configPath); err != nil {
		logs.Warnw("failed to load access log config", "path", configPath, "error", err)
	}
	return cfg
}
