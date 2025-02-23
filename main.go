package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
	"youtube_tracker/internal/api"
	mainHanler "youtube_tracker/internal/http_handler"
	"youtube_tracker/internal/youtube/stats"
)

type App struct {
	dbUrl      string
	httpServer *http.Server
}

func NewApp(ctx context.Context, host string, port int, dbUrl string) (*App, error) {
	channelRepository, err := stats.NewChannelRepository(ctx, dbUrl)
	if err != nil {
		return nil, err
	}

	httpHandler := mainHanler.NewHttpHandler(channelRepository)
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
		dbUrl:      dbUrl,
		httpServer: httpServer,
	}, nil
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

	//todo read params from a config file
	app, err := NewApp(
		ctx,
		"localhost",
		8080,
		"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	)

	if err != nil {
		log.Fatal(err)
	}

	app.Start()

	<-make(chan struct{})
}
