package main

import (
	"fmt"
	"strings"

	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

const configFileName = "sample.yaml"

type Config struct {
	Message       string
	DryRunMessage string
}

type fileConfig struct {
	Message       string `mapstructure:"message"`
	DryRunMessage string `mapstructure:"dry_run_message"`
}

var newConfigLoader = func() *pkgconfig.Loader {
	return pkgconfig.NewForApp("sample", configFileName)
}

func loadConfig() (Config, error) {
	config := defaultConfig()

	var loaded fileConfig
	if err := newConfigLoader().Load(&loaded); err != nil {
		if pkgconfig.IsConfigNotFound(err) {
			return config, nil
		}

		return Config{}, fmt.Errorf("load sample config: %w", err)
	}

	if value := strings.TrimSpace(loaded.Message); value != "" {
		config.Message = value
	}

	if value := strings.TrimSpace(loaded.DryRunMessage); value != "" {
		config.DryRunMessage = value
	}

	return config, nil
}

func defaultConfig() Config {
	return Config{
		Message:       "Hello MY CLI",
		DryRunMessage: "Dry run: would print Hello MY CLI",
	}
}
