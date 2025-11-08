package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"youtube_tracker/internal/api"
	mainHanler "youtube_tracker/internal/http_handler"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"

	"github.com/jackc/pgx/v5/pgxpool"
)

const numWorkers = 10

const (
	defaultPoolMaxSize               = 10
	defaultPoolMinSize               = 2
	defaultPoolMaxConnectionIdleTime = 30 * time.Minute
	defaultPoolMaxConnectionLifetime = time.Hour
	defaultPoolHealthCheckPeriod     = time.Minute
)

type App struct {
	dbUrl      string
	httpServer *http.Server
}

func NewApp(ctx context.Context, port int, dbURL string, authDir string, ytClient youtube.Client) (*App, error) {
	dbPool, err := createDbPool(ctx, dbURL)
	if err != nil {
		return nil, err
	}

	channelRepository := stats.NewChannelRepository(dbPool)

	channelService := stats.NewYoutubeChannelService(dbPool, channelRepository)

	var client youtube.Client
	if ytClient != nil {
		client = ytClient
	} else {
		httpClient, err := youtube.NewHttpClient(ctx, authDir)
		if err != nil {
			return nil, err
		}

		client = httpClient
	}

	workers := make([]youtube.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = youtube.NewStatisticsWorker(channelRepository, client)
	}

	collector := youtube.NewStatisticsCollector(channelRepository, workers, 10)
	httpHandler := mainHanler.NewHTTPHandler(collector, channelService, channelRepository)

	srv, err := api.NewServer(httpHandler)
	if err != nil {
		return nil, err
	}

	httpServer := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		dbUrl:      dbURL,
		httpServer: httpServer,
	}, nil
}

func createDbPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to parse connection string: %v\n", err)
		return nil, err
	}

	config.MaxConns = toInt32(getEnv("POOL_MAX_SIZE"), defaultPoolMaxSize)
	config.MinConns = toInt32(getEnv("POOL_MIN_SIZE"), defaultPoolMinSize)
	config.MaxConnLifetime = toDuration(getEnv("POOL_MAX_CONN_LIFETIME"), defaultPoolMaxConnectionLifetime)
	config.MaxConnIdleTime = toDuration(getEnv("POOL_MAX_CONN_IDLE_TIME"), defaultPoolMaxConnectionIdleTime)
	config.HealthCheckPeriod = toDuration(getEnv("POOL_HEALTH_CHECK_PERIOD"), defaultPoolHealthCheckPeriod)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func (app *App) Start() {
	log.Printf("Starting server on %s", app.httpServer.Addr)

	go func() {
		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
}

func (app *App) Stop(ctx context.Context) error {
	return app.httpServer.Shutdown(ctx)
}

func toInt32(value string, fallback int32) int32 {
	if len(value) == 0 {
		return fallback
	}

	if intValue, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(intValue)
	}

	return fallback
}

func toDuration(value string, fallback time.Duration) time.Duration {
	if len(value) == 0 {
		return fallback
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	return fallback
}

func getEnv(key string) string {
	return getEnvWithFallback(key, "")
}

func getEnvWithFallback(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	ctx := context.Background()

	portStr := getEnvWithFallback("APP_PORT", "8080")
	dbUrl := getEnvWithFallback("DB_URL", "")
	authDir := getEnvWithFallback("AUTH_DIR", "")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port specified: %v", err)
	}

	app, err := NewApp(ctx, port, dbUrl, authDir, nil)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	// Wait indefinitely
	<-make(chan struct{})
}
