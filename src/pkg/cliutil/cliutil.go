package cliutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func NormalizeLongFlags(args []string, flagNames ...string) []string {
	normalized := make([]string, 0, len(args))

	for _, arg := range args {
		if converted, ok := normalizeLongFlag(arg, flagNames); ok {
			normalized = append(normalized, converted)
			continue
		}

		normalized = append(normalized, arg)
	}

	return normalized
}

func ReadSingleInput(args []string, stdin io.Reader) (string, error) {
	switch len(args) {
	case 0:
		if isInteractiveInput(stdin) {
			return "", errors.New("json input is required as a single argument or via stdin")
		}

		content, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}

		raw := strings.TrimSpace(string(content))
		if raw == "" {
			return "", errors.New("json input is empty")
		}

		return raw, nil
	case 1:
		raw := strings.TrimSpace(args[0])
		if raw == "" {
			return "", errors.New("json input is empty")
		}

		return raw, nil
	default:
		return "", errors.New("accepts at most one JSON argument")
	}
}

func DecodeStrictJSON[T any](raw string) (T, error) {
	var value T

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode json input: %w", err)
	}

	if decoder.More() {
		return value, errors.New("json input must contain exactly one object")
	}

	return value, nil
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write json output: %w", err)
	}

	return nil
}

func normalizeLongFlag(arg string, flagNames []string) (string, bool) {
	for _, name := range flagNames {
		prefix := "-" + strings.TrimSpace(name)

		switch {
		case arg == prefix, strings.HasPrefix(arg, prefix+"="):
			return "-" + arg, true
		}
	}

	return "", false
}

func isInteractiveInput(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
