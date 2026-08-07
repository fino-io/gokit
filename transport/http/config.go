package http

import (
	"strings"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

const (
	EnvelopStyle = "envelope"
	AIPStyle     = "aip"

	underScoreEnvelopStyle = "_envelope"
	underScoreAIPStyle     = "_aip"
)

type Config struct {
	Style    string         `json:"style"` // default, aip, envelope
	Envelope EnvelopeConfig `json:"envelope"`
}

func (c *Config) GetStyle() string {
	if c != nil {
		return c.Style
	}
	return ""
}

func (c *Config) GetEnvelop() *EnvelopeConfig {
	if c != nil {
		return &c.Envelope
	}
	return &EnvelopeConfig{}
}

func NewConfig(path ...string) *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "transport.http"); err != nil {
		logs.Errorw("failed to get the http.transport config.", "path", strings.Join(path, "."), "error", err)
		return nil
	}
	return cfg
}
