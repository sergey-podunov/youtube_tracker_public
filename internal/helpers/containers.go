package helpers

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	_, filename, _, _ := runtime.Caller(0)
	srcRoot := filepath.Dir(filepath.Dir(filename))

	testcontainers.Logger = log.New(os.Stdout, "testcontainers: ", log.LstdFlags)

	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithInitScripts(
			filepath.Join(srcRoot, "..", "database", "schema.sql"),
			filepath.Join(srcRoot, "..", "testdata", "init-db.sql"),
		),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)),
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		return nil, err
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
	}, nil
}
