package app

import (
	"os"
	"path/filepath"
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
