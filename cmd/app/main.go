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

const PoolMaxSize = 10
const PoolMinSize = 2
const PoolMaxConnectionIdleTime = 30 * time.Minute
const PoolMaxConnectionLifetime = time.Hour
const PoolHealthCheckPeriod = time.Minute

type App struct {
	dbUrl      string
	httpServer *http.Server
}

func NewApp(ctx context.Context, host string, port int, dbURL string, authDir string, ytClient youtube.Client) (*App, error) {
	dbPool, err := createDbPool(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	
	channelRepository, err := stats.NewChannelRepository(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	
	channelService := stats.NewYoutubeChannelService(dbPool, channelRepository)
	
	var client youtube.Client
	if ytClient != nil {
		client = ytClient
	} else {
		httpCli, err := youtube.NewHttpClient(ctx, authDir)
		if err != nil {
			return nil, err
		}
		
		client = httpCli
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
		Addr:         host + ":" + strconv.Itoa(port),
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
	
	config.MaxConns = PoolMaxSize
	config.MinConns = PoolMinSize
	config.MaxConnLifetime = PoolMaxConnectionLifetime
	config.MaxConnIdleTime = PoolMaxConnectionIdleTime
	config.HealthCheckPeriod = PoolHealthCheckPeriod
	
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}
	
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return  nil, err
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

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func main() {
	ctx := context.Background()

	host := getEnv("APP_HOST", "")
	portStr := getEnv("APP_PORT", "8080")
	dbUrl := getEnv("DB_URL", "")
	authDir := getEnv("AUTH_DIR", "")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port specified: %v", err)
	}

	app, err := NewApp(
		ctx,
		host,
		port,
		dbUrl,
		authDir,
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	// Wait indefinitely
	<-make(chan struct{})
}
