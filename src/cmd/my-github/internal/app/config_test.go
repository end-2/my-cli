package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
github:
  base_url: https://example.github.local/api/v3
  token: "{{ .GITHUB_TOKEN }}"
  timeout: 3s
  user_agent: custom-agent
`)

	t.Setenv("GITHUB_TOKEN", "configured-from-env")

	configPath := filepath.Join(tempDir, configFileName)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if config.BaseURL != "https://example.github.local/api/v3" {
		t.Fatalf("BaseURL = %q, want custom base URL", config.BaseURL)
	}

	if config.Token != "configured-from-env" {
		t.Fatalf("Token = %q, want %q", config.Token, "configured-from-env")
	}

	if config.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, 3*time.Second)
	}

	if config.UserAgent != "custom-agent" {
		t.Fatalf("UserAgent = %q, want %q", config.UserAgent, "custom-agent")
	}
}

func TestLoadConfigAppliesBaseURLSpecificConfig(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	tempDir := t.TempDir()
	tempHome := t.TempDir()

	t.Setenv("HOME", tempHome)
	t.Setenv("GITHUB_TOKEN", "github-token")
	t.Setenv("GHE_TOKEN", "enterprise-token")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	content := []byte(`
github:
  base_url: https://ghe.example.com/api/v3
  timeout: 3s
  user_agent: default-agent
  token: "{{ .GITHUB_TOKEN }}"
  by_base_url:
    - alias: github.com
      base_url: https://api.github.com/
      token: "{{ .GITHUB_TOKEN }}"
    - alias: example-ghe
      base_url: https://ghe.example.com/api/v3/
      token: "{{ .GHE_TOKEN }}"
      timeout: 30s
      user_agent: enterprise-agent
`)

	configPath := filepath.Join(tempDir, configFileName)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if config.BaseURL != "https://ghe.example.com/api/v3" {
		t.Fatalf("BaseURL = %q, want enterprise base URL", config.BaseURL)
	}

	if config.Token != "enterprise-token" {
		t.Fatalf("Token = %q, want %q", config.Token, "enterprise-token")
	}

	if config.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, 30*time.Second)
	}

	if config.UserAgent != "enterprise-agent" {
		t.Fatalf("UserAgent = %q, want %q", config.UserAgent, "enterprise-agent")
	}
}

func TestFileConfigToClientConfigAppliesDefaultBaseURLOverride(t *testing.T) {
	config, err := fileConfig{
		GitHub: gitHubConfig{
			Timeout: "3s",
			ByBaseURL: []gitHubBaseURLConfigEntry{
				{
					Alias:     "github.com",
					BaseURL:   "https://api.github.com",
					Token:     "configured-token",
					UserAgent: "override-agent",
				},
			},
		},
	}.toClientConfig()
	if err != nil {
		t.Fatalf("toClientConfig returned error: %v", err)
	}

	if config.BaseURL != "https://api.github.com/" {
		t.Fatalf("BaseURL = %q, want default base URL", config.BaseURL)
	}

	if config.Token != "configured-token" {
		t.Fatalf("Token = %q, want %q", config.Token, "configured-token")
	}

	if config.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, 3*time.Second)
	}

	if config.UserAgent != "override-agent" {
		t.Fatalf("UserAgent = %q, want %q", config.UserAgent, "override-agent")
	}
}

func TestFileConfigToClientConfigRejectsInvalidTimeout(t *testing.T) {
	_, err := fileConfig{
		GitHub: gitHubConfig{
			Timeout: "not-a-duration",
		},
	}.toClientConfig()
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want timeout parse error")
	}
}

func TestFileConfigToClientConfigRejectsInvalidBaseURLSpecificTimeout(t *testing.T) {
	_, err := fileConfig{
		GitHub: gitHubConfig{
			ByBaseURL: []gitHubBaseURLConfigEntry{
				{
					Alias:   "github.com",
					BaseURL: "https://api.github.com/",
					Timeout: "not-a-duration",
				},
			},
		},
	}.toClientConfig()
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want timeout parse error")
	}
}

func TestFileConfigToClientConfigUsesAliasInMatchedEntryError(t *testing.T) {
	_, err := fileConfig{
		GitHub: gitHubConfig{
			BaseURL: "https://api.github.com/",
			ByBaseURL: []gitHubBaseURLConfigEntry{
				{
					Alias:   "github.com",
					BaseURL: "https://api.github.com/",
					Timeout: "not-a-duration",
				},
			},
		},
	}.toClientConfig()
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want timeout parse error")
	}

	if !strings.Contains(err.Error(), `github.by_base_url["github.com"]`) {
		t.Fatalf("error = %q, want alias-based config path", err)
	}
}

func TestFileConfigToClientConfigRejectsDuplicateNormalizedBaseURLs(t *testing.T) {
	_, err := fileConfig{
		GitHub: gitHubConfig{
			BaseURL: "https://api.github.com",
			ByBaseURL: []gitHubBaseURLConfigEntry{
				{Alias: "github.com", BaseURL: "https://api.github.com"},
				{Alias: "public-mirror", BaseURL: "https://api.github.com/"},
			},
		},
	}.toClientConfig()
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want duplicate base URL error")
	}

	for _, alias := range []string{"github.com", "public-mirror"} {
		if !strings.Contains(err.Error(), alias) {
			t.Fatalf("error = %q, want alias %q", err, alias)
		}
	}
}
