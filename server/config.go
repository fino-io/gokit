package server

import (
	"strings"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

type Config struct {
	DebugAddr string `json:"debugAddr" yaml:"debugAddr" default:":20170"`
	HttpAddr  string `json:"httpAddr" yaml:"httpAddr" default:":20171"`
	GrpcAddr  string `json:"grpcAddr" yaml:"grpcAddr" default:":20172"`
}

func NewConfig(path ...string) *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "server"); err != nil {
		logs.Errorw("failed to get the server config from "+strings.Join(path, "."), "error", err)
		return nil
	}
	return cfg
}
