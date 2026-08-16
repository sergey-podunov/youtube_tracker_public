//go:build database

package redisclient

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePasswordFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "redis-pass")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestNewClientConnectsWithPasswordFromFile(t *testing.T) {
	ctx := context.Background()

	container, err := helpers.CreateRedisContainerWithPassword(ctx, "test-password")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	t.Setenv("REDIS_PASSWORD", writePasswordFile(t, "test-password\n"))
	password, err := helpers.GetSecret("REDIS_PASSWORD")
	require.NoError(t, err)

	client, err := NewClient(ctx, slog.Default(), Config{Addr: container.Addr, Password: password})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})

	assert.NoError(t, client.Ping(ctx).Err())
}

func TestNewClientFailsWithWrongPassword(t *testing.T) {
	ctx := context.Background()

	container, err := helpers.CreateRedisContainerWithPassword(ctx, "correct-password")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	_, err = NewClient(ctx, slog.Default(), Config{Addr: container.Addr, Password: "wrong-password"})

	assert.Error(t, err)
}

func TestNewClientFailsWhenServerUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := NewClient(ctx, slog.Default(), Config{Addr: "localhost:1", Password: "irrelevant"})

	assert.Error(t, err)
}
