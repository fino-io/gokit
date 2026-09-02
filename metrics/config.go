package metrics

import finoconfig "github.com/fino-io/finokit/config"

type config struct {
	Enable    bool   `json:"enable" default:"false"`
	Namespace string `json:"namespace"`
}

func loadConfig() (*config, error) {
	cfg := &config{}
	if err := finoconfig.ScanFrom(cfg, "metrics"); err != nil {
		return nil, err
	}
	return cfg, nil
}
