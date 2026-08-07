package tracing

import (
	"strings"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

type Config struct {
	Enable            bool    `json:"enable" yaml:"enable" default:"false"`
	Endpoint          string  `json:"endpoint" yaml:"endpoint" default:"localhost:4318"`
	SampleRatio       float64 `json:"sampleRatio" yaml:"sampleRatio" default:"1"`
	Environment       string  `json:"environment" yaml:"environment"`
	ServiceVersion    string  `json:"serviceVersion" yaml:"serviceVersion"`
	ServiceInstanceID string  `json:"serviceInstanceID" yaml:"serviceInstanceID"`
}

const DefaultOTLPEndpoint = "localhost:4318"

func NewConfig(path ...string) *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "tracing"); err != nil {
		logs.Errorw("failed to get the tracing config from "+strings.Join(path, "."), "error", err)
		return nil
	}
	return cfg
}
