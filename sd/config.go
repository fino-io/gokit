package sd

import (
	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"

	"github.com/fino-io/gokit/retry"
	"github.com/fino-io/gokit/sd/direct"
	"github.com/fino-io/gokit/sd/etcdv3"
	// "github.com/fino-io/gokit/sd/nacos"
)

type Config struct {
	Mode      string                    `json:"mode" yaml:"mode" db:"mode"`                // etcd, direct
	Transport string                    `json:"transport" yaml:"transport" db:"transport"` // http, grpc
	Url       string                    `json:"url" yaml:"url"`
	Retry     *retry.Config             `json:"retry" yaml:"retry" db:"retry"`
	EtcdV3    *etcdv3.Config            `json:"etcd" yaml:"etcd"`
	Direct    map[string]*direct.Config `json:"direct" yaml:"direct" db:"direct"`
	// Nacos     *nacos.Config             `json:"nacos" yaml:"nacos"`
}

func NewConfig() *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "sd"); err != nil {
		logs.Errorw("failed to get the sd config", "error", err)
		return nil
	}
	return cfg
}
