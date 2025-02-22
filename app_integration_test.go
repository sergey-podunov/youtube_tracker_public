package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
	"youtube_tracker/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

const host = "localhost"
const port = 8081

func TestMainHttpHandlerIntegration(t *testing.T) {
	ctx := context.Background()
	pgContainer := createPgContainer(ctx)
	defer pgContainer.Container.Terminate(ctx)

	app := createApp(host, port, pgContainer.ConnectionString)
	app.Start()

	url := fmt.Sprintf("http://%s:%d", host, port)
	checkIfServerStarted(url)

	t.Run("Test /status endpoint", func(t *testing.T) {
		resp, err := http.Get(url + "/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, string(body), "ms") // Check uptime is returned
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := app.Stop(ctx)
	require.NoError(t, err)

	log.Println("Test suite completed.")
}

func createPgContainer(ctx context.Context) *testhelpers.PostgresContainer {
	pgContainer, err := testhelpers.CreatePostgresContainer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return pgContainer
}

func createApp(host string, port int, dbUrl string) *App {
	app, err := NewApp(host, port, dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	return app
}

func checkIfServerStarted(url string) {
	for {
		resp, err := http.Get(url + "/status")
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("Server is ready to accept connections on %s", url)
			return
		}

		time.Sleep(1 * time.Second)
	}
}
