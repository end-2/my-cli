package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-github/internal/github"
	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

const configFileName = "my-github.yaml"

type fileConfig struct {
	GitHub gitHubConfig `mapstructure:"github"`
}

type gitHubConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

func LoadConfig() (github.ClientConfig, error) {
	var cfg fileConfig
	loader := pkgconfig.NewForApp("my-github", configFileName)

	if err := loader.Load(&cfg); err != nil {
		if pkgconfig.IsConfigNotFound(err) {
			return github.DefaultClientConfig(), nil
		}

		return github.ClientConfig{}, fmt.Errorf("load my-github config: %w", err)
	}

	clientConfig, err := cfg.toClientConfig()
	if err != nil {
		return github.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c fileConfig) toClientConfig() (github.ClientConfig, error) {
	clientConfig := github.DefaultClientConfig()

	if value := strings.TrimSpace(c.GitHub.BaseURL); value != "" {
		clientConfig.BaseURL = value
	}

	if value := strings.TrimSpace(c.GitHub.Token); value != "" {
		clientConfig.Token = value
	}

	if value := strings.TrimSpace(c.GitHub.UserAgent); value != "" {
		clientConfig.UserAgent = value
	}

	if value := strings.TrimSpace(c.GitHub.Timeout); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return github.ClientConfig{}, fmt.Errorf("parse github.timeout: %w", err)
		}

		clientConfig.Timeout = timeout
	}

	return clientConfig, nil
}
