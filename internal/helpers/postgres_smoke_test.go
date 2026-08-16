//go:build database

package helpers

import (
	"context"
	"testing"
)

func TestCreatePostgresContainerSmoke(t *testing.T) {
	ctx := context.Background()

	c, err := CreatePostgresContainer(ctx)
	if err != nil {
		t.Fatalf("CreatePostgresContainer failed: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	if c.ConnectionString == "" {
		t.Fatal("expected non-empty connection string")
	}
	t.Logf("postgres connection string: %s", c.ConnectionString)
}
