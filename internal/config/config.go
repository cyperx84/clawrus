package config

import (
	"os"
	"path/filepath"

	"github.com/cyperx84/clawrus/internal/types"
	"gopkg.in/yaml.v3"
)

// PresetPath returns the path to ~/.clawrus/presets.yaml
func PresetPath() string {
	return filepath.Join(ConfigDir(), "presets.yaml")
}

// LoadPresets loads ~/.clawrus/presets.yaml
func LoadPresets() (*types.PresetConfig, error) {
	p := PresetPath()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.PresetConfig{Presets: make(map[string]types.Preset)}, nil
		}
		return nil, err
	}
	var cfg types.PresetConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Presets == nil {
		cfg.Presets = make(map[string]types.Preset)
	}
	return &cfg, nil
}

// SavePresets writes ~/.clawrus/presets.yaml
func SavePresets(cfg *types.PresetConfig) error {
	p := PresetPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawrus")
}

func ConfigPath() string {
	if p := os.Getenv("CLAWRUS_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(ConfigDir(), "groups.yaml")
}

// MainConfigPath returns the path to ~/.clawrus/config.yaml
func MainConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// LoadMainConfig loads ~/.clawrus/config.yaml (gateway settings etc.)
func LoadMainConfig() (*types.ClawrusConfig, error) {
	p := MainConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.ClawrusConfig{}, nil
		}
		return nil, err
	}
	var cfg types.ClawrusConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
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
