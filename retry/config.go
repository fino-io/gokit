package retry

import (
	"strings"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

type Config struct {
	Enable  bool `json:"enable" yaml:"enable" default:"false"`
	Timeout int  `json:"timeout" yaml:"timeout" default:"1000"`
	Max     int  `json:"max" yaml:"max" default:"3"`
}

func NewConfig(path ...string) *Config {
	cfg := &Config{}
	if err := config.ScanFrom(&cfg, "retry"); err != nil {
		logs.Errorw("failed to get the retry config from "+strings.Join(path, "."), "error", err)
		return nil
	}
	return cfg
}
