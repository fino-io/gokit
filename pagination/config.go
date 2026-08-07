package pagination

import (
	"errors"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
)

var errNilConfig = errors.New("pagination config is nil")

type Config struct {
	EncodedKey string `json:"encodedKey" yaml:"encodedKey"`
}

func NewConfig() *Config {
	cfg := &Config{}
	if err := config.ScanFrom(cfg, "pagination"); err != nil {
		logs.Errorw("failed to get the pagination config", "error", err)
		return nil
	}
	return cfg
}

func New() (*CursorCodec, error) {
	return NewWithConfig(NewConfig())
}

func NewWithConfig(cfg *Config) (*CursorCodec, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	return NewCursorCodecFromBase64(cfg.EncodedKey)
}
