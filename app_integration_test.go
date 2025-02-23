package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ogen-go/ogen/conv"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

const host = "localhost"
const port = 8081

func TestMainHttpHandlerIntegration(t *testing.T) {
	ctx := context.Background()
	pgContainer := createPgContainer(ctx)
	defer pgContainer.Container.Terminate(ctx)

	app := createApp(ctx, host, port, pgContainer.ConnectionString)
	app.Start()

	url := fmt.Sprintf("http://%s:%d", host, port)
	checkIfServerStarted(url)

	t.Run("Test /status endpoint", func(t *testing.T) {
		resp, err := http.Get(url + "/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "ms") // Check uptime is returned
	})

	t.Run("Test /youtube/channel endpoint", func(t *testing.T) {
		channelPostResp := &api.YoutubeChannel{}
		respCode, respStatus := executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannel{
				Name:      "Test Channel",
				YoutubeID: "TestYoutubeID",
			},
			channelPostResp,
		)

		assert.Equal(t, http.StatusCreated, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel", channelPostResp.Name)
		assert.Equal(t, "TestYoutubeID", channelPostResp.YoutubeID)
		assert.NotEmpty(t, channelPostResp.CreatedAt.Value)
		assert.Greater(t, channelPostResp.ID.Value, int64(0))

		channelGetResp := api.YoutubeChannel{}
		respCode, respStatus = executeGet(t, url+"/youtube/channel/"+conv.Int64ToString(channelPostResp.ID.Value), &channelGetResp)

		assert.Equal(t, http.StatusOK, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel", channelGetResp.Name)
		assert.Equal(t, "TestYoutubeID", channelGetResp.YoutubeID)
		assert.Equal(t, channelPostResp.CreatedAt.Value, channelGetResp.CreatedAt.Value)
		assert.Equal(t, channelPostResp.ID.Value, channelGetResp.ID.Value)
	})

	t.Run("Test /youtube/channel not found", func(t *testing.T) {
		channelGetResp := api.YoutubeChannel{}
		respCode, respStatus := executeGet(t, url+"/youtube/channel/123456", &channelGetResp)

		assert.Equal(t, http.StatusNotFound, respCode, "Response code: %s", respStatus)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := app.Stop(ctx)
	require.NoError(t, err)

	log.Println("Test suite completed.")
}

func executeGet(t *testing.T, url string, u json.Unmarshaler) (int, string) {
	resp, err := http.Get(url)
	require.NoError(t, err)

	defer resp.Body.Close()
	unmarshalBody(t, resp.Body, u)

	return resp.StatusCode, resp.Status
}

func executePost(t *testing.T, url string, m json.Marshaler, u json.Unmarshaler) (int, string) {
	jsonData, err := m.MarshalJSON()
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", io.NopCloser(bytes.NewReader(jsonData)))
	require.NoError(t, err)

	defer resp.Body.Close()
	unmarshalBody(t, resp.Body, u)

	return resp.StatusCode, resp.Status
}

func unmarshalBody(t *testing.T, respBody io.ReadCloser, u json.Unmarshaler) {
	body, err := io.ReadAll(respBody)
	require.NoError(t, err)

	if len(body) > 0 {
		err = u.UnmarshalJSON(body)
		require.NoError(t, err)
	}
}

func createPgContainer(ctx context.Context) *testhelpers.PostgresContainer {
	pgContainer, err := testhelpers.CreatePostgresContainer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return pgContainer
}

func createApp(ctx context.Context, host string, port int, dbUrl string) *App {
	app, err := NewApp(ctx, host, port, dbUrl)
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
