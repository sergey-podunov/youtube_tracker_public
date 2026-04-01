package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/helpers"
	mainHandler "youtube_tracker/internal/http_handler"
	"youtube_tracker/internal/notify"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ogen-go/ogen/middleware"
)

const numWorkers = 10

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(slog.String("component", "Main"))

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

func NewApp(ctx context.Context, port int, dbURL string, googleApiKey string, allowedOrigin string, notifierConfig notify.NotifierConfig,
	ytClient youtube.Client) (*App, error) {
	logger.Info("Starting app", "dockerTag", helpers.GetBuildTag())

	dbPool, err := createDbPool(ctx, logger, dbURL)
	if err != nil {
		return nil, err
	}

	channelRepository := stats.NewChannelRepository(logger, dbPool)

	channelService := stats.NewYoutubeChannelService(logger, dbPool, channelRepository)

	var client youtube.Client
	if ytClient != nil {
		client = ytClient
	} else {
		notifier := notify.NewNotifier(logger, notifierConfig)
		httpClient, err := youtube.NewHttpClient(ctx, notifier, googleApiKey)
		if err != nil {
			return nil, err
		}

		client = httpClient
	}

	workers := make([]youtube.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = youtube.NewStatisticsWorker(logger, channelRepository, client)
	}

	collector := youtube.NewStatisticsCollector(logger, channelRepository, workers, 10)
	httpHandler := mainHandler.NewHTTPHandler(logger, collector, channelService, channelRepository)

	chainMiddleware := middleware.ChainMiddlewares(
		mainHandler.RequestIDGenerator,
		mainHandler.NewRequestLogger(logger),
	)

	srv, err := api.NewServer(httpHandler, api.WithMiddleware(chainMiddleware))
	if err != nil {
		return nil, err
	}

	corsHandler, err := mainHandler.NewCORSHandler(srv, allowedOrigin, logger.With(slog.String("component", "CORSHandler")))
	if err != nil {
		return nil, err
	}

	httpServer := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      corsHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		dbUrl:      dbURL,
		httpServer: httpServer,
	}, nil
}

func createDbPool(ctx context.Context, logger *slog.Logger, dbURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Unable to parse connection string", "error", err)
		return nil, err
	}

	config.MaxConns = helpers.ToInt32(helpers.GetEnv("POOL_MAX_SIZE"), defaultPoolMaxSize)
	config.MinConns = helpers.ToInt32(helpers.GetEnv("POOL_MIN_SIZE"), defaultPoolMinSize)
	config.MaxConnLifetime = helpers.ToDuration(helpers.GetEnv("POOL_MAX_CONN_LIFETIME"), defaultPoolMaxConnectionLifetime)
	config.MaxConnIdleTime = helpers.ToDuration(helpers.GetEnv("POOL_MAX_CONN_IDLE_TIME"), defaultPoolMaxConnectionIdleTime)
	config.HealthCheckPeriod = helpers.ToDuration(helpers.GetEnv("POOL_HEALTH_CHECK_PERIOD"), defaultPoolHealthCheckPeriod)

	logger.Info("Database connection config",
		slog.Group("config",
			"host", config.ConnConfig.Host,
			"port", config.ConnConfig.Port,
			"user", config.ConnConfig.User,
			"database", config.ConnConfig.Database,
			"maxCon", config.MaxConns,
			"minCon", config.MinConns,
			"maxConnLifetime", config.MaxConnLifetime.Milliseconds(),
			"maxConnIdleTime", config.MaxConnIdleTime.Milliseconds(),
			"healthCheckPeriod", config.HealthCheckPeriod.Milliseconds(),
		))

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Error("Unable to connect to database", "error", err)
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("Connected to database")
	return pool, nil
}

func (app *App) Start() {
	logger.Info("Starting server", "port", app.httpServer.Addr)

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
	dbUrl, err := helpers.GetRequiredEnv("DB_URL")
	if err != nil {
		log.Fatal(err)
	}
	googleApiKey, err := helpers.GetRequiredEnv("GOOGLE_API_KEY")
	if err != nil {
		log.Fatal(err)
	}
	allowedOrigin, err := helpers.GetRequiredEnv("ALLOWED_ORIGIN")
	if err != nil {
		log.Fatal(err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port specified: %v", err)
	}

	notifierConfig := notify.NotifierConfig{
		TelegramToken:   helpers.GetEnvWithFallback("TELEGRAM_TOKEN", ""),
		TelegramChatID:  helpers.ToInt64(helpers.GetEnv("TELEGRAM_CHAT_ID"), int64(0)),
		SlackWebhookURL: helpers.GetEnvWithFallback("SLACK_WEBHOOK_URL", ""),
	}

	app, err := NewApp(ctx, port, dbUrl, googleApiKey, allowedOrigin, notifierConfig, nil)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	// Wait indefinitely
	<-make(chan struct{})
}
