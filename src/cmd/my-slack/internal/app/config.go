package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-slack/internal/slack"
	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

const configFileName = "my-slack.yaml"

type fileConfig struct {
	Slack slackConfig `mapstructure:"slack"`
}

type slackConfig struct {
	BaseURL    string                      `mapstructure:"base_url"`
	Token      string                      `mapstructure:"token"`
	Timeout    string                      `mapstructure:"timeout"`
	UserAgent  string                      `mapstructure:"user_agent"`
	Workspaces []slackWorkspaceConfigEntry `mapstructure:"workspaces"`
}

type slackConfigEntry struct {
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

type slackWorkspaceConfigEntry struct {
	Alias     string `mapstructure:"alias"`
	BaseURL   string `mapstructure:"base_url"`
	Token     string `mapstructure:"token"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

func LoadConfig() (slack.ClientConfig, error) {
	return LoadConfigForRequest(slack.Request{})
}

func LoadConfigForRequest(request slack.Request) (slack.ClientConfig, error) {
	var cfg fileConfig
	loader := pkgconfig.NewForApp("my-slack", configFileName)

	if err := loader.Load(&cfg); err != nil {
		if pkgconfig.IsConfigNotFound(err) {
			return fileConfig{}.toClientConfig(request)
		}

		return slack.ClientConfig{}, fmt.Errorf("load my-slack config: %w", err)
	}

	clientConfig, err := cfg.toClientConfig(request)
	if err != nil {
		return slack.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c fileConfig) toClientConfig(request slack.Request) (slack.ClientConfig, error) {
	clientConfig := slack.DefaultClientConfig()

	if value := strings.TrimSpace(c.Slack.BaseURL); value != "" {
		clientConfig.BaseURL = value
	}

	if err := applySlackConfigEntry(&clientConfig, slackConfigEntry{
		Token:     c.Slack.Token,
		Timeout:   c.Slack.Timeout,
		UserAgent: c.Slack.UserAgent,
	}, "slack"); err != nil {
		return slack.ClientConfig{}, err
	}

	if value := strings.TrimSpace(request.BaseURL); value != "" {
		if _, err := slack.NormalizeBaseURL(value); err != nil {
			return slack.ClientConfig{}, fmt.Errorf("parse json input field \"base_url\": %w", err)
		}

		clientConfig.BaseURL = value
	}

	if alias := strings.TrimSpace(request.Alias); alias != "" {
		if err := c.Slack.applyWorkspaceAlias(&clientConfig, request, alias); err != nil {
			return slack.ClientConfig{}, err
		}
	}

	return clientConfig, nil
}

func (c slackConfig) applyWorkspaceAlias(clientConfig *slack.ClientConfig, request slack.Request, requestedAlias string) error {
	matchedIndex := -1

	for index, entry := range c.Workspaces {
		if strings.TrimSpace(entry.Alias) != requestedAlias {
			continue
		}

		if matchedIndex >= 0 {
			return fmt.Errorf(
				"duplicate slack.workspaces aliases for %q: %q and %q",
				requestedAlias,
				c.Workspaces[matchedIndex].descriptor(),
				entry.descriptor(),
			)
		}

		matchedIndex = index
	}

	if matchedIndex < 0 {
		return fmt.Errorf("json input field \"alias\" %q does not match any slack.workspaces entry", requestedAlias)
	}

	matchedEntry := c.Workspaces[matchedIndex]
	if value := strings.TrimSpace(matchedEntry.BaseURL); value != "" {
		normalizedBaseURL, err := slack.NormalizeBaseURL(value)
		if err != nil {
			return fmt.Errorf("parse %s.base_url: %w", matchedEntry.fieldPrefix(), err)
		}

		if requestedBaseURL := strings.TrimSpace(request.BaseURL); requestedBaseURL != "" {
			normalizedRequestedBaseURL, err := slack.NormalizeBaseURL(requestedBaseURL)
			if err != nil {
				return fmt.Errorf("parse json input field \"base_url\": %w", err)
			}

			if normalizedRequestedBaseURL != normalizedBaseURL {
				return fmt.Errorf(
					"json input fields \"alias\" and \"base_url\" must refer to the same slack.workspaces entry: alias %q uses %q, got %q",
					requestedAlias,
					matchedEntry.BaseURL,
					requestedBaseURL,
				)
			}
		}

		clientConfig.BaseURL = value
	}

	return applySlackConfigEntry(clientConfig, slackConfigEntry{
		Token:     matchedEntry.Token,
		Timeout:   matchedEntry.Timeout,
		UserAgent: matchedEntry.UserAgent,
	}, matchedEntry.fieldPrefix())
}

func (e slackWorkspaceConfigEntry) descriptor() string {
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

func (e slackWorkspaceConfigEntry) fieldPrefix() string {
	if alias := strings.TrimSpace(e.Alias); alias != "" {
		return fmt.Sprintf("slack.workspaces[%q]", alias)
	}

	return "slack.workspaces"
}

func applySlackConfigEntry(clientConfig *slack.ClientConfig, entry slackConfigEntry, fieldPrefix string) error {
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
