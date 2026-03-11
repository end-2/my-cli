package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkgconfig "github.com/end-2/my-cli/src/pkg/config"
)

func TestRootCommandPrintsGreeting(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, nil); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "Hello MY CLI\n" {
		t.Fatalf("stdout = %q, want %q", got, "Hello MY CLI\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsGreetingFromConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(&stdout, &stderr, nil, Dependencies{
		LoadConfig: func() (Config, error) {
			return Config{
				Message:       "Hello from config",
				DryRunMessage: "Dry run from config",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	if got := stdout.String(); got != "Hello from config\n" {
		t.Fatalf("stdout = %q, want %q", got, "Hello from config\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsVersion(t *testing.T) {
	originalVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, []string{"--version"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "1.2.3\n" {
		t.Fatalf("stdout = %q, want %q", got, "1.2.3\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsVersionWithSingleDashLongFlag(t *testing.T) {
	originalVersion := Version
	Version = "9.9.9"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, []string{"-version"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "9.9.9\n" {
		t.Fatalf("stdout = %q, want %q", got, "9.9.9\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsDryRunMessage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, []string{"--dry-run"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "Dry run: would print Hello MY CLI\n" {
		t.Fatalf("stdout = %q, want %q", got, "Dry run: would print Hello MY CLI\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsDryRunMessageFromConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(&stdout, &stderr, []string{"--dry-run"}, Dependencies{
		LoadConfig: func() (Config, error) {
			return Config{
				Message:       "Hello from config",
				DryRunMessage: "Dry run from config",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	if got := stdout.String(); got != "Dry run from config\n" {
		t.Fatalf("stdout = %q, want %q", got, "Dry run from config\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, []string{"--help"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"sample is a small Cobra-based CLI example for my-cli.",
		"--dry-run",
		"--version",
		"sample.yaml",
	} {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Fatalf("stdout = %q, want to contain %q", output, expected)
		}
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestLoadConfigReadsSampleConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(`
message: Hello from file
`)), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	originalNewConfigLoader := newConfigLoader
	newConfigLoader = func() *pkgconfig.Loader {
		return pkgconfig.New(configPath)
	}
	t.Cleanup(func() {
		newConfigLoader = originalNewConfigLoader
	})

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if config.Message != "Hello from file" {
		t.Fatalf("config.Message = %q, want %q", config.Message, "Hello from file")
	}

	if config.DryRunMessage != "Dry run: would print Hello MY CLI" {
		t.Fatalf("config.DryRunMessage = %q, want default value", config.DryRunMessage)
	}
}

func TestRootCommandPrintsHelpWithSingleDashLongFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(&stdout, &stderr, []string{"-help"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got == "" {
		t.Fatal("stdout is empty, want help output")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
