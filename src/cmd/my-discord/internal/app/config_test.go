package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-discord/internal/discord"
)

func TestLoadConfigLoadsCurrentDirectoryFile(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	tempDir := t.TempDir()
	tempHome := t.TempDir()

	t.Setenv("HOME", tempHome)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	content := []byte(`
discord:
  base_url: https://discord.example.com/api/v10/
  token: "{{ .DISCORD_BOT_TOKEN }}"
  token_type: Bearer
  timeout: 3s
  user_agent: custom-agent
`)

	t.Setenv("DISCORD_BOT_TOKEN", "configured-from-env")

	configPath := filepath.Join(tempDir, configFileName)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if config.BaseURL != "https://discord.example.com/api/v10/" {
		t.Fatalf("BaseURL = %q, want custom base URL", config.BaseURL)
	}

	if config.Token != "configured-from-env" {
		t.Fatalf("Token = %q, want %q", config.Token, "configured-from-env")
	}

	if config.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want %q", config.TokenType, "Bearer")
	}

	if config.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, 3*time.Second)
	}

	if config.UserAgent != "custom-agent" {
		t.Fatalf("UserAgent = %q, want %q", config.UserAgent, "custom-agent")
	}
}

func TestFileConfigToClientConfigUsesBotAliasOverride(t *testing.T) {
	config, err := fileConfig{
		Discord: discordConfig{
			BaseURL:   "https://discord.com/api/v10/",
			Token:     "default-token",
			TokenType: "Bot",
			Timeout:   "3s",
			UserAgent: "default-agent",
			Bots: []discordBotConfigEntry{
				{
					Alias:     "bot-dev",
					Token:     "dev-token",
					UserAgent: "dev-agent",
				},
				{
					Alias:     "bot-prod",
					BaseURL:   "https://discord.example.com/api/v10/",
					Token:     "prod-token",
					TokenType: "Bearer",
					Timeout:   "30s",
					UserAgent: "prod-agent",
				},
			},
		},
	}.toClientConfig(discord.Request{Alias: "bot-prod"})
	if err != nil {
		t.Fatalf("toClientConfig returned error: %v", err)
	}

	if config.BaseURL != "https://discord.example.com/api/v10/" {
		t.Fatalf("BaseURL = %q, want alias base URL", config.BaseURL)
	}

	if config.Token != "prod-token" {
		t.Fatalf("Token = %q, want %q", config.Token, "prod-token")
	}

	if config.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want %q", config.TokenType, "Bearer")
	}

	if config.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, 30*time.Second)
	}

	if config.UserAgent != "prod-agent" {
		t.Fatalf("UserAgent = %q, want %q", config.UserAgent, "prod-agent")
	}
}

func TestFileConfigToClientConfigRejectsAliasAndBaseURLMismatch(t *testing.T) {
	_, err := fileConfig{
		Discord: discordConfig{
			Bots: []discordBotConfigEntry{
				{
					Alias:   "bot-prod",
					BaseURL: "https://discord.example.com/api/v10/",
				},
			},
		},
	}.toClientConfig(discord.Request{
		Alias:   "bot-prod",
		BaseURL: "https://discord.other.example/api/v10/",
	})
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want alias/base_url mismatch")
	}

	if !strings.Contains(err.Error(), `"alias" and "base_url" must refer to the same`) {
		t.Fatalf("error = %v, want alias/base_url mismatch", err)
	}
}
