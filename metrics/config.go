package metrics

import "github.com/fino-io/finokit/config"

type Config struct {
	Enable    bool   `json:"enable" yaml:"enable" default:"false"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

func loadConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.ScanFrom(cfg, "metrics"); err != nil {
		return nil, err
	}
	return cfg, nil
}
