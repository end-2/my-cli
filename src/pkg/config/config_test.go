package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sampleConfig struct {
	App struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	Feature struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"feature"`
}

func TestLoadFile(t *testing.T) {
	path := writeConfigFile(t, t.TempDir(), DefaultFileName, `
app:
  name: my-cli
  port: 8080
log:
  level: info
`)

	var cfg sampleConfig
	if err := LoadFile(path, &cfg); err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.App.Name != "my-cli" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "my-cli")
	}

	if cfg.App.Port != 8080 {
		t.Fatalf("App.Port = %d, want %d", cfg.App.Port, 8080)
	}

	if cfg.Log.Level != "info" {
		t.Fatalf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoadFileRendersEnvironmentVariables(t *testing.T) {
	t.Setenv("APP_NAME", "templated-cli")
	t.Setenv("APP_PORT", "8081")

	path := writeConfigFile(t, t.TempDir(), DefaultFileName, `
app:
  name: {{ .APP_NAME }}
  port: {{ .APP_PORT }}
`)

	var cfg sampleConfig
	if err := LoadFile(path, &cfg); err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.App.Name != "templated-cli" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "templated-cli")
	}

	if cfg.App.Port != 8081 {
		t.Fatalf("App.Port = %d, want %d", cfg.App.Port, 8081)
	}
}

func TestLoaderLoadMergesExplicitFileWithHighestPriority(t *testing.T) {
	currentDir := t.TempDir()
	homeDir := t.TempDir()
	etcDir := t.TempDir()
	explicitDir := t.TempDir()
	explicitPath := writeConfigFile(t, explicitDir, DefaultFileName, `
app:
  name: from-explicit
  port: 9000
log:
  level: error
`)

	writeConfigFile(t, filepath.Join(etcDir, "sample"), DefaultFileName, `
app:
  name: from-etc
  port: 7000
log:
  level: warn
feature:
  enabled: true
`)

	writeConfigFile(t, homeDir, DefaultFileName, `
app:
  port: 8000
log:
  level: info
`)

	writeConfigFile(t, currentDir, DefaultFileName, `
app:
  name: from-current
`)

	loader := New(explicitPath)
	loader.appName = "sample"
	loader.currentDir = func() (string, error) { return currentDir, nil }
	loader.homeDir = func() (string, error) { return homeDir, nil }
	loader.etcDir = etcDir

	var cfg sampleConfig
	if err := loader.Load(&cfg); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Name != "from-explicit" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "from-explicit")
	}

	if cfg.App.Port != 9000 {
		t.Fatalf("App.Port = %d, want %d", cfg.App.Port, 9000)
	}

	if cfg.Log.Level != "error" {
		t.Fatalf("Log.Level = %q, want %q", cfg.Log.Level, "error")
	}

	if !cfg.Feature.Enabled {
		t.Fatal("Feature.Enabled = false, want true")
	}
}

func TestLoaderLoadMergesConfigsByPriority(t *testing.T) {
	currentDir := t.TempDir()
	homeDir := t.TempDir()
	etcDir := t.TempDir()

	t.Setenv("APP_NAME", "from-home-template")

	writeConfigFile(t, filepath.Join(etcDir, "sample"), DefaultFileName, `
app:
  name: from-etc
  port: 7000
log:
  level: warn
feature:
  enabled: true
`)

	writeConfigFile(t, homeDir, DefaultFileName, `
app:
  port: 8000
  name: {{ .APP_NAME }}
log:
  level: info
`)

	writeConfigFile(t, currentDir, DefaultFileName, `
app:
  name: from-current
`)

	loader := NewForApp("sample", DefaultFileName)
	loader.currentDir = func() (string, error) { return currentDir, nil }
	loader.homeDir = func() (string, error) { return homeDir, nil }
	loader.etcDir = etcDir

	var cfg sampleConfig
	if err := loader.Load(&cfg); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Name != "from-current" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "from-current")
	}

	if cfg.App.Port != 8000 {
		t.Fatalf("App.Port = %d, want %d", cfg.App.Port, 8000)
	}

	if cfg.Log.Level != "info" {
		t.Fatalf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}

	if !cfg.Feature.Enabled {
		t.Fatal("Feature.Enabled = false, want true")
	}
}

func TestLoadFileReturnsErrorForMissingEnvironmentVariable(t *testing.T) {
	path := writeConfigFile(t, t.TempDir(), DefaultFileName, `
app:
  name: {{ .MISSING_ENV }}
`)

	var cfg sampleConfig
	err := LoadFile(path, &cfg)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want missing environment variable error")
	}

	if !strings.Contains(err.Error(), "render config template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoaderLoadUsesExecutableNameWhenAppNameIsEmpty(t *testing.T) {
	etcDir := t.TempDir()
	writeConfigFile(t, filepath.Join(etcDir, "sample-bin"), DefaultFileName, `
app:
  name: from-executable
`)

	loader := NewForApp("", DefaultFileName)
	loader.currentDir = func() (string, error) { return t.TempDir(), nil }
	loader.homeDir = func() (string, error) { return t.TempDir(), nil }
	loader.executablePath = func() (string, error) { return "/usr/local/bin/sample-bin", nil }
	loader.etcDir = etcDir

	var cfg sampleConfig
	if err := loader.Load(&cfg); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Name != "from-executable" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "from-executable")
	}
}

func TestLoaderLoadReturnsErrorWhenNoConfigFilesExist(t *testing.T) {
	loader := NewForApp("sample", DefaultFileName)
	loader.currentDir = func() (string, error) { return t.TempDir(), nil }
	loader.homeDir = func() (string, error) { return t.TempDir(), nil }
	loader.etcDir = t.TempDir()

	var cfg sampleConfig
	err := loader.Load(&cfg)
	if err == nil {
		t.Fatal("Load returned nil error, want missing config error")
	}

	if !strings.Contains(err.Error(), "no config file found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileReturnsErrorWhenConfigFileDoesNotExist(t *testing.T) {
	var cfg sampleConfig
	err := LoadFile(filepath.Join(t.TempDir(), "missing.yaml"), &cfg)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want missing file error")
	}

	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileReturnsErrorForInvalidTarget(t *testing.T) {
	path := writeConfigFile(t, t.TempDir(), DefaultFileName, `
app:
  name: invalid-target
  port: 7000
`)

	var cfg sampleConfig
	err := LoadFile(path, cfg)
	if !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("error = %v, want ErrInvalidTarget", err)
	}
}

func writeConfigFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return path
}
