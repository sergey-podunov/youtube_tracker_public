//go:build database

package helpers

import (
	"context"
	"testing"
)

func TestCreateRedisContainerSmoke(t *testing.T) {
	ctx := context.Background()

	c, err := CreateRedisContainer(ctx)
	if err != nil {
		t.Fatalf("CreateRedisContainer failed: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	if c.ConnectionString == "" {
		t.Fatal("expected non-empty connection string")
	}
	t.Logf("redis connection string: %s", c.ConnectionString)
}
