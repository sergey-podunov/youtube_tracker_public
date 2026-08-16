package helpers

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
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
			filepath.Join(srcRoot, "..", "testdata", "schema.sql"),
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

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable", "search_path=youtube_tracker")
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
	}, nil
}

type RedisContainer struct {
	*redis.RedisContainer
	ConnectionString string
	Addr             string
}

func CreateRedisContainer(ctx context.Context) (*RedisContainer, error) {
	return CreateRedisContainerWithPassword(ctx, "")
}

// CreateRedisContainerWithPassword starts a Redis container with requirepass
// set to password. Addr on the returned container holds host:port.
func CreateRedisContainerWithPassword(ctx context.Context, password string) (*RedisContainer, error) {
	testcontainers.Logger = log.New(os.Stdout, "testcontainers: ", log.LstdFlags)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second),
				wait.ForListeningPort("6379/tcp"),
			),
		),
	}
	if password != "" {
		opts = append(opts, testcontainers.CustomizeRequestOption(func(req *testcontainers.GenericContainerRequest) error {
			if len(req.Cmd) == 0 {
				req.Cmd = []string{"redis-server"}
			}
			req.Cmd = append(req.Cmd, "--requirepass", password)
			return nil
		}))
	}

	redisContainer, err := redis.Run(ctx, "redis:8.10.0-alpine", opts...)
	if err != nil {
		return nil, err
	}

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	host, err := redisContainer.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return nil, err
	}

	return &RedisContainer{
		RedisContainer:   redisContainer,
		ConnectionString: connStr,
		Addr:             net.JoinHostPort(host, port.Port()),
	}, nil
}

func StartPgAndGetConnection(ctx context.Context) (*pgx.Conn, error) {
	pgContainer, err := CreatePostgresContainer(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := connectWithTrace(ctx, pgContainer.ConnectionString)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

type stdLogger struct{}

func (l stdLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	log.Printf("[pgx %s] %s - %v", level, msg, data)
}

func connectWithTrace(ctx context.Context, connStr string) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	if os.Getenv("TRACE_SQL") == "true" {
		cfg.Tracer = &tracelog.TraceLog{
			Logger:   stdLogger{},
			LogLevel: tracelog.LogLevelTrace,
		}
	}

	return pgx.ConnectConfig(ctx, cfg)
}
