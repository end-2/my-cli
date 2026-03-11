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
	BaseURL   string                     `mapstructure:"base_url"`
	Token     string                     `mapstructure:"token"`
	Timeout   string                     `mapstructure:"timeout"`
	UserAgent string                     `mapstructure:"user_agent"`
	ByBaseURL []gitHubBaseURLConfigEntry `mapstructure:"by_base_url"`
}

type gitHubConfigEntry struct {
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

type gitHubBaseURLConfigEntry struct {
	Alias     string `mapstructure:"alias"`
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

	if err := applyGitHubConfigEntry(&clientConfig, gitHubConfigEntry{
		Token:     c.GitHub.Token,
		Timeout:   c.GitHub.Timeout,
		UserAgent: c.GitHub.UserAgent,
	}, "github"); err != nil {
		return github.ClientConfig{}, err
	}

	if err := c.GitHub.applyBaseURLOverride(&clientConfig); err != nil {
		return github.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c gitHubConfig) applyBaseURLOverride(clientConfig *github.ClientConfig) error {
	if len(c.ByBaseURL) == 0 {
		return nil
	}

	selectedBaseURL, err := github.NormalizeBaseURL(clientConfig.BaseURL)
	if err != nil {
		return fmt.Errorf("parse github.base_url: %w", err)
	}

	matchedIndex := -1

	for index, entry := range c.ByBaseURL {
		normalizedBaseURL, err := github.NormalizeBaseURL(entry.BaseURL)
		if err != nil {
			if alias := strings.TrimSpace(entry.Alias); alias != "" {
				return fmt.Errorf("parse github.by_base_url[%d].base_url for alias %q: %w", index, alias, err)
			}

			return fmt.Errorf("parse github.by_base_url[%d].base_url: %w", index, err)
		}

		if normalizedBaseURL != selectedBaseURL {
			continue
		}

		if matchedIndex >= 0 {
			return fmt.Errorf(
				"duplicate github.by_base_url entries for %q: %q and %q",
				selectedBaseURL,
				c.ByBaseURL[matchedIndex].descriptor(),
				entry.descriptor(),
			)
		}

		matchedIndex = index
	}

	if matchedIndex < 0 {
		return nil
	}

	matchedEntry := c.ByBaseURL[matchedIndex]

	return applyGitHubConfigEntry(clientConfig, gitHubConfigEntry{
		Token:     matchedEntry.Token,
		Timeout:   matchedEntry.Timeout,
		UserAgent: matchedEntry.UserAgent,
	}, matchedEntry.fieldPrefix())
}

func (e gitHubBaseURLConfigEntry) descriptor() string {
	alias := strings.TrimSpace(e.Alias)
	baseURL := strings.TrimSpace(e.BaseURL)

	switch {
	case alias == "":
		return baseURL
	case baseURL == "":
		return alias
	default:
		return fmt.Sprintf("%s (%s)", alias, baseURL)
	}
}

func (e gitHubBaseURLConfigEntry) fieldPrefix() string {
	if alias := strings.TrimSpace(e.Alias); alias != "" {
		return fmt.Sprintf("github.by_base_url[%q]", alias)
	}

	return fmt.Sprintf("github.by_base_url[%q]", e.BaseURL)
}

func applyGitHubConfigEntry(clientConfig *github.ClientConfig, entry gitHubConfigEntry, fieldPrefix string) error {
	if value := strings.TrimSpace(entry.Token); value != "" {
		clientConfig.Token = value
	}

	if value := strings.TrimSpace(entry.UserAgent); value != "" {
		clientConfig.UserAgent = value
	}

	if value := strings.TrimSpace(entry.Timeout); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse %s.timeout: %w", fieldPrefix, err)
		}

		clientConfig.Timeout = timeout
	}

	return nil
}
