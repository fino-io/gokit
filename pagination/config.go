package pagination

import (
	"errors"

	finoconfig "github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/finokit/pagination"
)

var errNilConfig = errors.New("pagination config is nil")

type config struct {
	EncodedKey string `json:"encodedKey" yaml:"encodedKey"`
}

func loadConfig() *config {
	cfg := &config{}
	if err := finoconfig.ScanFrom(cfg, "pagination"); err != nil {
		logs.Errorw("failed to get the pagination config", "error", err)
		return nil
	}
	return cfg
}

// New loads the legacy global pagination configuration.
func New() (*CursorCodec, error) {
	return newWithConfig(loadConfig())
}

func newWithConfig(cfg *config) (*CursorCodec, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	return pagination.NewCursorCodecFromBase64(cfg.EncodedKey)
}
