package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-discord/internal/discord"
	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

const configFileName = "my-discord.yaml"

type fileConfig struct {
	Discord discordConfig `mapstructure:"discord"`
}

type discordConfig struct {
	BaseURL   string                  `mapstructure:"base_url"`
	Token     string                  `mapstructure:"token"`
	TokenType string                  `mapstructure:"token_type"`
	Timeout   string                  `mapstructure:"timeout"`
	UserAgent string                  `mapstructure:"user_agent"`
	Bots      []discordBotConfigEntry `mapstructure:"bots"`
}

type discordConfigEntry struct {
	Token     string `mapstructure:"token"`
	TokenType string `mapstructure:"token_type"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

type discordBotConfigEntry struct {
	Alias     string `mapstructure:"alias"`
	BaseURL   string `mapstructure:"base_url"`
	Token     string `mapstructure:"token"`
	TokenType string `mapstructure:"token_type"`
	Timeout   string `mapstructure:"timeout"`
	UserAgent string `mapstructure:"user_agent"`
}

func LoadConfig() (discord.ClientConfig, error) {
	return LoadConfigForRequest(discord.Request{})
}

func LoadConfigForRequest(request discord.Request) (discord.ClientConfig, error) {
	var cfg fileConfig
	loader := pkgconfig.NewForApp("my-discord", configFileName)

	if err := loader.Load(&cfg); err != nil {
		if pkgconfig.IsConfigNotFound(err) {
			return fileConfig{}.toClientConfig(request)
		}

		return discord.ClientConfig{}, fmt.Errorf("load my-discord config: %w", err)
	}

	clientConfig, err := cfg.toClientConfig(request)
	if err != nil {
		return discord.ClientConfig{}, err
	}

	return clientConfig, nil
}

func (c fileConfig) toClientConfig(request discord.Request) (discord.ClientConfig, error) {
	clientConfig := discord.DefaultClientConfig()

	if value := strings.TrimSpace(c.Discord.BaseURL); value != "" {
		clientConfig.BaseURL = value
	}

	if err := applyDiscordConfigEntry(&clientConfig, discordConfigEntry{
		Token:     c.Discord.Token,
		TokenType: c.Discord.TokenType,
		Timeout:   c.Discord.Timeout,
		UserAgent: c.Discord.UserAgent,
	}, "discord"); err != nil {
		return discord.ClientConfig{}, err
	}

	if value := strings.TrimSpace(request.BaseURL); value != "" {
		if _, err := discord.NormalizeBaseURL(value); err != nil {
			return discord.ClientConfig{}, fmt.Errorf("parse json input field \"base_url\": %w", err)
		}

		clientConfig.BaseURL = value
	}

	if alias := strings.TrimSpace(request.Alias); alias != "" {
		if err := c.Discord.applyBotAlias(&clientConfig, request, alias); err != nil {
			return discord.ClientConfig{}, err
		}
	}

	return clientConfig, nil
}

func (c discordConfig) applyBotAlias(clientConfig *discord.ClientConfig, request discord.Request, requestedAlias string) error {
	matchedIndex := -1

	for index, entry := range c.Bots {
		if strings.TrimSpace(entry.Alias) != requestedAlias {
			continue
		}

		if matchedIndex >= 0 {
			return fmt.Errorf(
				"duplicate discord.bots aliases for %q: %q and %q",
				requestedAlias,
				c.Bots[matchedIndex].descriptor(),
				entry.descriptor(),
			)
		}

		matchedIndex = index
	}

	if matchedIndex < 0 {
		return fmt.Errorf("json input field \"alias\" %q does not match any discord.bots entry", requestedAlias)
	}

	matchedEntry := c.Bots[matchedIndex]
	if value := strings.TrimSpace(matchedEntry.BaseURL); value != "" {
		normalizedBaseURL, err := discord.NormalizeBaseURL(value)
		if err != nil {
			return fmt.Errorf("parse %s.base_url: %w", matchedEntry.fieldPrefix(), err)
		}

		if requestedBaseURL := strings.TrimSpace(request.BaseURL); requestedBaseURL != "" {
			normalizedRequestedBaseURL, err := discord.NormalizeBaseURL(requestedBaseURL)
			if err != nil {
				return fmt.Errorf("parse json input field \"base_url\": %w", err)
			}

			if normalizedRequestedBaseURL != normalizedBaseURL {
				return fmt.Errorf(
					"json input fields \"alias\" and \"base_url\" must refer to the same discord.bots entry: alias %q uses %q, got %q",
					requestedAlias,
					matchedEntry.BaseURL,
					requestedBaseURL,
				)
			}
		}

		clientConfig.BaseURL = value
	}

	return applyDiscordConfigEntry(clientConfig, discordConfigEntry{
		Token:     matchedEntry.Token,
		TokenType: matchedEntry.TokenType,
		Timeout:   matchedEntry.Timeout,
		UserAgent: matchedEntry.UserAgent,
	}, matchedEntry.fieldPrefix())
}

func (e discordBotConfigEntry) descriptor() string {
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

func (e discordBotConfigEntry) fieldPrefix() string {
	if alias := strings.TrimSpace(e.Alias); alias != "" {
		return fmt.Sprintf("discord.bots[%q]", alias)
	}

	return "discord.bots"
}

func applyDiscordConfigEntry(clientConfig *discord.ClientConfig, entry discordConfigEntry, fieldPrefix string) error {
	if value := strings.TrimSpace(entry.Token); value != "" {
		clientConfig.Token = value
	}

	if value := strings.TrimSpace(entry.TokenType); value != "" {
		tokenType, err := discord.NormalizeTokenType(value)
		if err != nil {
			return fmt.Errorf("parse %s.token_type: %w", fieldPrefix, err)
		}

		clientConfig.TokenType = tokenType
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
