package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-prom/internal/prom"
	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

const configFileName = "my-prom.yaml"

type fileConfig struct {
	Prometheus promConfig `mapstructure:"prometheus"`
}

type promConfig struct {
	BaseURL   string                    `mapstructure:"base_url"`
	Token     string                    `mapstructure:"token"`
	Timeout   string                    `mapstructure:"timeout"`
	UserAgent string                    `mapstructure:"user_agent"`
	Instances []promInstanceConfigEntry `mapstructure:"instances"`
}

type promConfigEntry struct {
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

type promInstanceConfigEntry struct {
	Alias     string `mapstructure:"alias"`
	BaseURL   string `mapstructure:"base_url"`
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

func LoadConfig() (prom.ClientConfig, error) {
	return LoadConfigForRequest(prom.Request{})
}

func LoadConfigForRequest(request prom.Request) (prom.ClientConfig, error) {
	var cfg fileConfig
	loader := pkgconfig.NewForApp("my-prom", configFileName)

	if err := loader.Load(&cfg); err != nil {
		if pkgconfig.IsConfigNotFound(err) {
			return fileConfig{}.toClientConfig(request)
		}

		return prom.ClientConfig{}, fmt.Errorf("load my-prom config: %w", err)
	}

	clientConfig, err := cfg.toClientConfig(request)
	if err != nil {
		return prom.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c fileConfig) toClientConfig(request prom.Request) (prom.ClientConfig, error) {
	clientConfig := prom.DefaultClientConfig()

	if value := strings.TrimSpace(c.Prometheus.BaseURL); value != "" {
		clientConfig.BaseURL = value
	}

	if value := strings.TrimSpace(request.BaseURL); value != "" {
		if _, err := prom.NormalizeBaseURL(value); err != nil {
			return prom.ClientConfig{}, fmt.Errorf("parse json input field \"base_url\": %w", err)
		}

		clientConfig.BaseURL = value
	}

	if err := applyPromConfigEntry(&clientConfig, promConfigEntry{
		Token:     c.Prometheus.Token,
		Timeout:   c.Prometheus.Timeout,
		UserAgent: c.Prometheus.UserAgent,
	}, "prometheus"); err != nil {
		return prom.ClientConfig{}, err
	}

	if err := c.Prometheus.applyInstanceOverride(&clientConfig, request); err != nil {
		return prom.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c promConfig) applyInstanceOverride(clientConfig *prom.ClientConfig, request prom.Request) error {
	if alias := strings.TrimSpace(request.Alias); alias != "" {
		return c.applyAliasOverride(clientConfig, request, alias)
	}

	return c.applySelectedBaseURLOverride(clientConfig, strings.TrimSpace(request.BaseURL) != "")
}

func (c promConfig) applyAliasOverride(clientConfig *prom.ClientConfig, request prom.Request, requestedAlias string) error {
	matchedIndex := -1
	matchedBaseURL := ""

	for index, entry := range c.Instances {
		normalizedBaseURL, err := prom.NormalizeBaseURL(entry.BaseURL)
		if err != nil {
			if alias := strings.TrimSpace(entry.Alias); alias != "" {
				return fmt.Errorf("parse prometheus.instances[%d].base_url for alias %q: %w", index, alias, err)
			}

			return fmt.Errorf("parse prometheus.instances[%d].base_url: %w", index, err)
		}

		if strings.TrimSpace(entry.Alias) != requestedAlias {
			continue
		}

		if matchedIndex >= 0 {
			return fmt.Errorf(
				"duplicate prometheus.instances aliases for %q: %q and %q",
				requestedAlias,
				c.Instances[matchedIndex].descriptor(),
				entry.descriptor(),
			)
		}

		matchedIndex = index
		matchedBaseURL = normalizedBaseURL
	}

	if matchedIndex < 0 {
		return fmt.Errorf("json input field \"alias\" %q does not match any prometheus.instances entry", requestedAlias)
	}

	if value := strings.TrimSpace(request.BaseURL); value != "" {
		requestedBaseURL, err := prom.NormalizeBaseURL(value)
		if err != nil {
			return fmt.Errorf("parse json input field \"base_url\": %w", err)
		}

		if requestedBaseURL != matchedBaseURL {
			return fmt.Errorf(
				"json input fields \"alias\" and \"base_url\" must refer to the same prometheus.instances entry: alias %q uses %q, got %q",
				requestedAlias,
				c.Instances[matchedIndex].BaseURL,
				value,
			)
		}
	}

	matchedEntry := c.Instances[matchedIndex]
	clientConfig.BaseURL = strings.TrimSpace(matchedEntry.BaseURL)

	return applyPromConfigEntry(clientConfig, promConfigEntry{
		Token:     matchedEntry.Token,
		Timeout:   matchedEntry.Timeout,
		UserAgent: matchedEntry.UserAgent,
	}, matchedEntry.fieldPrefix())
}

func (c promConfig) applySelectedBaseURLOverride(clientConfig *prom.ClientConfig, requestBaseURLSelected bool) error {
	if len(c.Instances) == 0 {
		return nil
	}

	selectedBaseURL, err := prom.NormalizeBaseURL(clientConfig.BaseURL)
	if err != nil {
		if requestBaseURLSelected {
			return fmt.Errorf("parse json input field \"base_url\": %w", err)
		}

		return fmt.Errorf("parse prometheus.base_url: %w", err)
	}

	matchedIndex := -1

	for index, entry := range c.Instances {
		normalizedBaseURL, err := prom.NormalizeBaseURL(entry.BaseURL)
		if err != nil {
			if alias := strings.TrimSpace(entry.Alias); alias != "" {
				return fmt.Errorf("parse prometheus.instances[%d].base_url for alias %q: %w", index, alias, err)
			}

			return fmt.Errorf("parse prometheus.instances[%d].base_url: %w", index, err)
		}

		if normalizedBaseURL != selectedBaseURL {
			continue
		}

		if matchedIndex >= 0 {
			return fmt.Errorf(
				"duplicate prometheus.instances entries for %q: %q and %q",
				selectedBaseURL,
				c.Instances[matchedIndex].descriptor(),
				entry.descriptor(),
			)
		}

		matchedIndex = index
	}

	if matchedIndex < 0 {
		return nil
	}

	matchedEntry := c.Instances[matchedIndex]

	return applyPromConfigEntry(clientConfig, promConfigEntry{
		Token:     matchedEntry.Token,
		Timeout:   matchedEntry.Timeout,
		UserAgent: matchedEntry.UserAgent,
	}, matchedEntry.fieldPrefix())
}

func (e promInstanceConfigEntry) descriptor() string {
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

func (e promInstanceConfigEntry) fieldPrefix() string {
	if alias := strings.TrimSpace(e.Alias); alias != "" {
		return fmt.Sprintf("prometheus.instances[%q]", alias)
	}

	return fmt.Sprintf("prometheus.instances[%q]", e.BaseURL)
}

func applyPromConfigEntry(clientConfig *prom.ClientConfig, entry promConfigEntry, fieldPrefix string) error {
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
