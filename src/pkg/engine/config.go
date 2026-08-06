package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = "goro-config.json"

type Config struct {
	ManifestURL string `json:"manifest_url"`
	ExeName     string `json:"exe_name"`
}

func DefaultConfig() *Config {
	return &Config{
		ManifestURL: "",
		ExeName:     "ragexe.exe",
	}
}

func LoadConfig(dir string) (*Config, error) {
	path := filepath.Join(dir, configFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(dir string, cfg *Config) error {
	path := filepath.Join(dir, configFileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
