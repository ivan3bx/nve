package nve

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Versioning bool `yaml:"versioning"`
}

func defaultConfig() *Config {
	return &Config{Versioning: true}
}

func LoadConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig()
	}

	path := filepath.Join(home, ".config", "nve", "config.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			writeDefaultConfig(path, cfg)
			return cfg
		}

		log.Printf("[WARN] failed to read config %s: %v; using defaults", path, err)
		return defaultConfig()
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Printf("[WARN] failed to parse %s: %v", path, err)
		return defaultConfig()
	}

	return cfg
}

func writeDefaultConfig(path string, cfg *Config) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[WARN] failed to create config directory %s: %v", dir, err)
		return
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Printf("[WARN] failed to marshal default config: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[WARN] failed to write default config to %s: %v", path, err)
	}
}
