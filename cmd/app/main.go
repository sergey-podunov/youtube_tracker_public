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
	"youtube_tracker/internal/redisclient"
	"youtube_tracker/internal/redisdebug"
	"youtube_tracker/internal/storage"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/ytclient"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ogen-go/ogen/middleware"
	"github.com/redis/go-redis/v9"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(slog.String("component", "Main"))

const (
	defaultPoolMaxSize               = 10
	defaultPoolMinSize               = 2
	defaultPoolMaxConnectionIdleTime = 30 * time.Minute
	defaultPoolMaxConnectionLifetime = time.Hour
	defaultPoolHealthCheckPeriod     = time.Minute
)

type App struct {
	dbUrl       string
	httpServer  *http.Server
	redisClient *redis.Client
}

func NewApp(ctx context.Context, port int, dbURL string, googleApiKey string, allowedOrigin string, notifierConfig notify.NotifierConfig,
	redisConfig *redisclient.Config, ytClient ytclient.Client) (*App, error) {
	logger.Info("Starting app", "dockerTag", helpers.GetBuildTag())

	dbPool, err := createDbPool(ctx, logger, dbURL)
	if err != nil {
		return nil, err
	}

	channelRepository := stats.NewChannelRepository(logger, dbPool)
	videoRepository := stats.NewVideoRepository(logger, dbPool)

	videoService := stats.NewYoutubeVideoService(logger, dbPool, videoRepository, channelRepository)

	var client ytclient.Client
	if ytClient != nil {
		client = ytClient
	} else {
		notifier := notify.NewNotifier(logger, notifierConfig)
		httpClient, err := ytclient.NewHttpClient(ctx, notifier, googleApiKey)
		if err != nil {
			return nil, err
		}

		client = httpClient
	}

	var fileFetcher storage.FileFetcher
	if s3Endpoint := helpers.GetEnv("S3_ENDPOINT"); s3Endpoint != "" {
		s3AccessKey := helpers.GetEnvWithFallback("S3_ACCESS_KEY", "")
		s3SecretKey := helpers.GetEnvWithFallback("S3_SECRET_KEY", "")
		s3Bucket := helpers.GetEnvWithFallback("S3_BUCKET", "thumbnails")

		if s3AccessKey == "" || s3SecretKey == "" {
			logger.Warn("S3 credentials not configured, storage disabled")
		} else {
			minioClient, err := minio.New(s3Endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
				Secure: false,
			})
			if err != nil {
				return nil, err
			}

			s3Storage := storage.NewS3StorageMinio(logger, minioClient, s3Bucket)
			fileFetcher = storage.NewFileFetcher(logger, s3Storage, &http.Client{Timeout: 30 * time.Second})
			logger.Info("S3 storage enabled", "endpoint", s3Endpoint, "bucket", s3Bucket)
		}
	}

	var redisClient *redis.Client
	if redisConfig != nil {
		redisClient, err = redisclient.NewClient(ctx, logger, *redisConfig)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Redis not configured, skipping")
	}

	channelService := stats.NewYoutubeChannelService(logger, dbPool, channelRepository, client, fileFetcher)

	collector := youtube.NewStatisticsCollector(logger)

	channelWorker := youtube.NewChannelWorker(logger, channelRepository, client, 50)
	channelJobController := youtube.NewWorkerJobController(channelWorker)

	videoWorker := youtube.NewVideoWorker(logger, videoRepository, client, 50)
	videoJobController := youtube.NewWorkerJobController(videoWorker)

	videoDiscoveryWorker := youtube.NewVideoDiscoveryWorker(logger, channelRepository, videoService, client, 50, 10)
	videoDiscoveryJobController := youtube.NewWorkerJobController(videoDiscoveryWorker)

	httpHandler := mainHandler.NewHTTPHandler(logger, collector, channelJobController, videoJobController, videoDiscoveryJobController, channelService, videoService, channelRepository, videoRepository)

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

	// rootHandler serves the application via corsHandler. When Redis is
	// configured, the temporary /redis/* debug endpoints are mounted in
	// front of it; everything else falls through to the app.
	var rootHandler = corsHandler
	if redisClient != nil {
		mux := http.NewServeMux()
		redisdebug.NewHandler(logger.With(slog.String("component", "RedisDebug")), redisClient).Register(mux)
		mux.Handle("/", corsHandler)
		rootHandler = mux
	}

	httpServer := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      rootHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		dbUrl:       dbURL,
		httpServer:  httpServer,
		redisClient: redisClient,
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
	if app.redisClient != nil {
		if err := app.redisClient.Close(); err != nil {
			logger.Warn("Failed to close Redis client", "error", err)
		}
	}

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

	var redisConfig *redisclient.Config
	if redisAddr := helpers.GetEnv("REDIS_ADDR"); redisAddr != "" {
		redisPassword, err := helpers.GetSecret("REDIS_PASSWORD")
		if err != nil {
			log.Fatal(err)
		}

		redisConfig = &redisclient.Config{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       int(helpers.ToInt32(helpers.GetEnv("REDIS_DB"), 0)),
		}
	}

	app, err := NewApp(ctx, port, dbUrl, googleApiKey, allowedOrigin, notifierConfig, redisConfig, nil)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	// Wait indefinitely
	<-make(chan struct{})
}
