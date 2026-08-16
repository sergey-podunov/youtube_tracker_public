package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempSecret(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestGetSecretPlainValue(t *testing.T) {
	t.Setenv("TEST_SECRET", "plain-value")

	value, err := GetSecret("TEST_SECRET")

	require.NoError(t, err)
	assert.Equal(t, "plain-value", value)
}

func TestGetSecretFromFile(t *testing.T) {
	t.Setenv("TEST_SECRET", writeTempSecret(t, "file-value"))

	value, err := GetSecret("TEST_SECRET")

	require.NoError(t, err)
	assert.Equal(t, "file-value", value)
}

func TestGetSecretTrimsTrailingNewline(t *testing.T) {
	t.Setenv("TEST_SECRET", writeTempSecret(t, "file-value\n"))

	value, err := GetSecret("TEST_SECRET")

	require.NoError(t, err)
	assert.Equal(t, "file-value", value)
}

func TestGetSecretMissingFileFallsBackToValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TEST_SECRET", path)

	value, err := GetSecret("TEST_SECRET")

	require.NoError(t, err)
	assert.Equal(t, path, value)
}

func TestGetSecretUnreadablePathErrors(t *testing.T) {
	// a directory exists but cannot be read as a file
	t.Setenv("TEST_SECRET", t.TempDir())

	_, err := GetSecret("TEST_SECRET")

	assert.Error(t, err)
}

func TestGetSecretNothingSet(t *testing.T) {
	value, err := GetSecret("TEST_SECRET_THAT_IS_NEVER_SET")

	require.NoError(t, err)
	assert.Empty(t, value)
}
