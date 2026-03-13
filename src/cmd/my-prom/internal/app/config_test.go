package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/end-2/my-cli/src/cmd/my-prom/internal/prom"
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
prometheus:
  base_url: https://prom.example.com/
  token: "{{ .PROM_TOKEN }}"
  timeout: 3s
  user_agent: custom-agent
`)

	t.Setenv("PROM_TOKEN", "configured-from-env")

	configPath := filepath.Join(tempDir, configFileName)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if config.BaseURL != "https://prom.example.com/" {
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

func TestLoadConfigAppliesInstanceSpecificConfig(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	tempDir := t.TempDir()
	tempHome := t.TempDir()

	t.Setenv("HOME", tempHome)
	t.Setenv("PROM_DEV_TOKEN", "dev-token")
	t.Setenv("PROM_PROD_TOKEN", "prod-token")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	content := []byte(`
prometheus:
  base_url: https://prom.prod.example.com/
  timeout: 3s
  user_agent: default-agent
  token: "{{ .PROM_DEV_TOKEN }}"
  instances:
    - alias: dev-prom
      base_url: https://prom.dev.example.com/
      token: "{{ .PROM_DEV_TOKEN }}"
    - alias: prod-prom
      base_url: https://prom.prod.example.com/
      token: "{{ .PROM_PROD_TOKEN }}"
      timeout: 30s
      user_agent: prod-agent
`)

	configPath := filepath.Join(tempDir, configFileName)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if config.BaseURL != "https://prom.prod.example.com/" {
		t.Fatalf("BaseURL = %q, want production base URL", config.BaseURL)
	}

	if config.Token != "prod-token" {
		t.Fatalf("Token = %q, want %q", config.Token, "prod-token")
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
		Prometheus: promConfig{
			Instances: []promInstanceConfigEntry{
				{
					Alias:   "prod-prom",
					BaseURL: "https://prom.prod.example.com/",
				},
			},
		},
	}.toClientConfig(prom.Request{
		Alias:   "prod-prom",
		BaseURL: "https://prom.other.example.com/",
	})
	if err == nil {
		t.Fatal("toClientConfig returned nil error, want alias/base_url mismatch")
	}

	if !strings.Contains(err.Error(), `"alias" and "base_url" must refer to the same`) {
		t.Fatalf("error = %v, want alias/base_url mismatch", err)
	}
}
