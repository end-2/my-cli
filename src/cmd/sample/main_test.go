package main

import (
	"bytes"
	"testing"
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
	} {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Fatalf("stdout = %q, want to contain %q", output, expected)
		}
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
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
