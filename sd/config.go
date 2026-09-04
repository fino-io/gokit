package sd

import (
	finoconfig "github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"

	"github.com/fino-io/gokit/sd/direct"
	"github.com/fino-io/gokit/sd/etcdv3"
)

type config struct {
	Mode      string                    `json:"mode" yaml:"mode" db:"mode"`                // etcd, direct
	Transport string                    `json:"transport" yaml:"transport" db:"transport"` // http, grpc
	Url       string                    `json:"url" yaml:"url"`
	EtcdV3    *etcdv3.Config            `json:"etcd" yaml:"etcd"`
	Direct    map[string]*direct.Config `json:"direct" yaml:"direct" db:"direct"`
}

func loadConfig() *config {
	cfg := &config{}
	if err := finoconfig.ScanFrom(cfg, "sd"); err != nil {
		logs.Errorw("failed to get the sd config", "error", err)
		return nil
	}
	return cfg
}
