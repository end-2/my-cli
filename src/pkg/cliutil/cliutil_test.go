package cliutil

import (
	"bytes"
	"strings"
	"testing"
)

func TestNormalizeLongFlags(t *testing.T) {
	args := NormalizeLongFlags([]string{"-version", "-help", "-dry-run", "payload"}, "version", "help", "dry-run")

	expected := []string{"--version", "--help", "--dry-run", "payload"}
	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}

	for index := range expected {
		if args[index] != expected[index] {
			t.Fatalf("args[%d] = %q, want %q", index, args[index], expected[index])
		}
	}
}

func TestReadSingleInputFromArgument(t *testing.T) {
	raw, err := ReadSingleInput([]string{`{"kind":"issue"}`}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ReadSingleInput returned error: %v", err)
	}

	if raw != `{"kind":"issue"}` {
		t.Fatalf("raw = %q, want %q", raw, `{"kind":"issue"}`)
	}
}

func TestReadSingleInputFromStdin(t *testing.T) {
	raw, err := ReadSingleInput(nil, strings.NewReader(" \n {\"kind\":\"issue\"}\n"))
	if err != nil {
		t.Fatalf("ReadSingleInput returned error: %v", err)
	}

	if raw != `{"kind":"issue"}` {
		t.Fatalf("raw = %q, want %q", raw, `{"kind":"issue"}`)
	}
}

func TestDecodeStrictJSONRejectsUnknownFields(t *testing.T) {
	type payload struct {
		Kind string `json:"kind"`
	}

	_, err := DecodeStrictJSON[payload](`{"kind":"issue","unknown":true}`)
	if err == nil {
		t.Fatal("DecodeStrictJSON returned nil error, want unknown field error")
	}
}

func TestWriteJSON(t *testing.T) {
	var stdout bytes.Buffer

	err := WriteJSON(&stdout, map[string]string{"kind": "issue"})
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "\"kind\": \"issue\"") {
		t.Fatalf("stdout = %q, want pretty json output", got)
	}
}
