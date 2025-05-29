package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"github.com/param108/gosiege/siege"
)

func ParseConfig(path string) (*siege.SiegeConfig, error) {
	// This function should parse the configuration file at the given path
	// and return a SiegeConfig object or an error if parsing fails.
	// Config is in json format.
	// Open the file, read its contents, and unmarshal it into a SiegeConfig struct.
	cfgFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	data, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &siege.SiegeConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return cfg, nil
}
