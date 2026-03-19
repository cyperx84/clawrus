package config

import (
	"os"
	"path/filepath"

	"github.com/cyperx84/threadgroups/internal/types"
	"gopkg.in/yaml.v3"
)

func ConfigPath() string {
	if p := os.Getenv("THREADGROUPS_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".threadgroups", "groups.yaml")
}

func Load() (*types.GroupConfig, error) {
	p := ConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.GroupConfig{Groups: make(map[string]types.Group)}, nil
		}
		return nil, err
	}
	var cfg types.GroupConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string]types.Group)
	}
	return &cfg, nil
}

func Save(cfg *types.GroupConfig) error {
	p := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
