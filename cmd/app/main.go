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
	"youtube_tracker/internal/helpers"
	mainHanler "youtube_tracker/internal/http_handler"
	"youtube_tracker/internal/notify"
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

func NewApp(ctx context.Context, port int, dbURL string, authDir string, notifierConfig notify.NotifierConfig, ytClient youtube.Client) (*App, error) {
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
		notifier := notify.NewNotifier(notifierConfig)
		httpClient, err := youtube.NewHttpClient(ctx, notifier, authDir)
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

	config.MaxConns = helpers.ToInt32(helpers.GetEnv("POOL_MAX_SIZE"), defaultPoolMaxSize)
	config.MinConns = helpers.ToInt32(helpers.GetEnv("POOL_MIN_SIZE"), defaultPoolMinSize)
	config.MaxConnLifetime = helpers.ToDuration(helpers.GetEnv("POOL_MAX_CONN_LIFETIME"), defaultPoolMaxConnectionLifetime)
	config.MaxConnIdleTime = helpers.ToDuration(helpers.GetEnv("POOL_MAX_CONN_IDLE_TIME"), defaultPoolMaxConnectionIdleTime)
	config.HealthCheckPeriod = helpers.ToDuration(helpers.GetEnv("POOL_HEALTH_CHECK_PERIOD"), defaultPoolHealthCheckPeriod)

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

func main() {
	ctx := context.Background()

	portStr := helpers.GetEnvWithFallback("APP_PORT", "8080")
	dbUrl := helpers.GetEnvWithFallback("DB_URL", "")
	authDir := helpers.GetEnvWithFallback("AUTH_DIR", "")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port specified: %v", err)
	}

	notifierConfig := notify.NotifierConfig{
		TelegramToken:   helpers.GetEnvWithFallback("TELEGRAM_TOKEN", ""),
		TelegramChatID:  helpers.ToInt64(helpers.GetEnv("TELEGRAM_CHAT_ID"), int64(0)),
		SlackWebhookURL: helpers.GetEnvWithFallback("SLACK_WEBHOOK_URL", ""),
	}
	
	app, err := NewApp(ctx, port, dbUrl, authDir, notifierConfig,  nil)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	// Wait indefinitely
	<-make(chan struct{})
}
