package server

import (
	"strings"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

type Perf struct {
	MaxProcess int `json:"maxProcess"`
}

type Config struct {
	DebugAddr string `json:"debugAddr" yaml:"debugAddr" default:":20170"`
	HttpAddr  string `json:"httpAddr" yaml:"httpAddr" default:":20171"`
	GrpcAddr  string `json:"grpcAddr" yaml:"grpcAddr" default:":20172"`
	TcpAddr   string `json:"tcpAddr" yaml:"tpcAddr" default:":20173"`
	UdpAddr   string `json:"udpAddr" yaml:"udpAddr" default:":20174"`
	Perf      Perf   `json:"Perf" yaml:"Perf"`
}

func NewConfig(path ...string) *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "server"); err != nil {
		logs.Errorw("failed to get the server config from "+strings.Join(path, "."), "error", err)
		return nil
	}
	return cfg
}
