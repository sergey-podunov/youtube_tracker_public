package helpers

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func GetBuildTag() string {
	return GetEnvWithFallback("BUILD_TAG", "undefined")
}

func GetEnv(key string) string {
	return GetEnvWithFallback(key, "")
}

func GetEnvWithFallback(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetRequiredEnv(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}

	return "", fmt.Errorf("%s environment variable is required", key)
}

// GetSecret returns the value for key, allowing it to be supplied via a
// file: if the value of <key> is a path to an existing file, the file's
// contents are returned with surrounding whitespace trimmed. Otherwise the
// value itself is returned ("" if unset). A file that exists but cannot be
// read is an error.
//
// When the value looks like a filesystem path but no such file exists, a
// warning is logged before falling back to the literal value — this usually
// means an intended secret file is missing (e.g. an unmounted volume). The
// path itself is not logged, to avoid leaking infrastructure details.
func GetSecret(key string) (string, error) {
	value := GetEnv(key)
	if value == "" {
		return "", nil
	}

	content, err := os.ReadFile(value)
	if errors.Is(err, os.ErrNotExist) {
		if strings.ContainsRune(value, '/') {
			slog.Warn("secret value looks like a file path but no such file exists; using it as a literal value",
				"key", key)
		}
		return value, nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %s from file %s: %w", key, value, err)
	}

	return strings.TrimSpace(string(content)), nil
}
