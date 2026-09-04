package accesslog

import (
	"time"

	finoconfig "github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

const configPath = "transport.accessLog"

type config struct {
	SlowThreshold time.Duration `json:"slowThreshold"`
	SampleEvery   uint64        `json:"sampleEvery"`
	HTTP          httpConfig    `json:"http"`
	GRPC          grpcConfig    `json:"grpc"`
}

type httpConfig struct {
	SkipPaths []string `json:"skipPaths"`
}

type grpcConfig struct {
	SkipMethods []string `json:"skipMethods"`
}

func defaultConfig() config {
	return config{
		SlowThreshold: 500 * time.Millisecond,
		SampleEvery:   100,
		HTTP: httpConfig{SkipPaths: []string{
			"/healthz",
			"/readyz",
			"/metrics",
		}},
		GRPC: grpcConfig{SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		}},
	}
}

func loadConfig() config {
	cfg := defaultConfig()
	if err := finoconfig.ScanFrom(&cfg, configPath); err != nil {
		logs.Warnw("failed to load access log config", "path", configPath, "error", err)
	}
	return cfg
}
