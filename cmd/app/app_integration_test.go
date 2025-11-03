//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube"

	"github.com/ogen-go/ogen/conv"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

const host = "localhost"
const port = 8081

type MockYoutubeClient struct {
	GetChannelIdFunc   func(channelName string) (string, error)
	GetChannelDataFunc func(channelId string) (*youtube.ChannelData, error)
}

func (m *MockYoutubeClient) GetChannelId(channelName string) (string, error) {
	if m.GetChannelIdFunc != nil {
		return m.GetChannelIdFunc(channelName)
	}
	return "", fmt.Errorf("GetChannelId not implemented in mock")
}

func (m *MockYoutubeClient) GetChannelData(channelId string) (*youtube.ChannelData, error) {
	if m.GetChannelDataFunc != nil {
		return m.GetChannelDataFunc(channelId)
	}
	return nil, fmt.Errorf("GetChannelData not implemented in mock")
}

var mockClient = &MockYoutubeClient{
	GetChannelIdFunc: func(channelName string) (string, error) {
		if channelName == "Test Channel" {
			return "TestYoutubeID", nil
		}
		return "", fmt.Errorf("channel not found: %s", channelName)
	},
	GetChannelDataFunc: func(channelId string) (*youtube.ChannelData, error) {
		if channelId == "TestYoutubeID" {
			return &youtube.ChannelData{
				ChannelID:        "TestYoutubeID",
				SubscribersCount: 151617,
			}, nil
		}
		return nil, fmt.Errorf("channel data not found for ID: %s", channelId)
	},
}

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

	t.Run("/youtube/channel posts conflict", func(t *testing.T) {
		executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannel{
				Name:      "Test Channel conflict",
				YoutubeID: "TestYoutubeID_conflict",
			},
			nil,
		)

		channelErrorResp := &api.YoutubeChannelPostConflict{}
		respCode, respStatus := executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannel{
				Name:      "Test Channel conflict",
				YoutubeID: "TestYoutubeID_conflict",
			},
			channelErrorResp,
		)

		assert.Equal(t, http.StatusConflict, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel conflict", channelErrorResp.Name)
		assert.Equal(t, "TestYoutubeID_conflict", channelErrorResp.YoutubeID)
	})

	t.Run("Test /youtube/channel not found", func(t *testing.T) {
		channelGetResp := api.YoutubeChannel{}
		respCode, respStatus := executeGet(t, url+"/youtube/channel/123456", &channelGetResp)

		assert.Equal(t, http.StatusNotFound, respCode, "Response code: %s", respStatus)
	})

	t.Run("Test youtube channel statistics", func(t *testing.T) {
		channelPostResp := &api.YoutubeChannel{}
		_, _ = executePost(t, url+"/youtube/channel", &api.YoutubeChannel{Name: "Test Channel", YoutubeID: "TestYoutubeID"}, channelPostResp)

		generationStartedResp := api.StatGenerationStarted{}
		respCode, respStatus := executePost(t, url+"/schedule", nil, &generationStartedResp)

		assert.Equal(t, http.StatusOK, respCode, "Response code: %s", respStatus)
		jobStatusPath := generationStartedResp.StatusPath
		assert.Regexp(t, `^/schedule/job/\d+$`, jobStatusPath, "StatusPath should match /schedule/job/{id} format")

		time.Sleep(time.Second * 2)

		jobStatusResp := api.JobStatus{}
		respCode, respStatus = executeGet(t, url+jobStatusPath, &jobStatusResp)

		assert.Equal(t, http.StatusOK, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "COMPLETE", string(jobStatusResp.Status))
		assert.GreaterOrEqual(t, jobStatusResp.Total.Value, int32(1))
		assert.GreaterOrEqual(t, jobStatusResp.Ready.Value, int32(1))
		assert.GreaterOrEqual(t, jobStatusResp.Error.Value, int32(0))

		channelStatisticsResp := api.YoutubeChannelStatistics{}
		respCode, respStatus = executeGet(t,
			url+"/youtube/channel/"+conv.Int64ToString(channelPostResp.ID.Value)+"/statistics", &channelStatisticsResp)

		assert.Equal(t, http.StatusOK, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "TestYoutubeID", channelStatisticsResp.YoutubeID.Value)
		assert.Equal(t, channelPostResp.ID.Value, channelStatisticsResp.ID)
		assert.Equal(t, 1, len(channelStatisticsResp.Statistics))

		channelStatistic := channelStatisticsResp.Statistics[0]

		assert.Equal(t, int64(151617), channelStatistic.Subscribers)
		assert.False(t, channelStatistic.Date.IsZero())
	})
	
	t.Run("Test /youtube/channel statistics not found", func(t *testing.T) {
		respCode, respStatus := executeGet(t, url+"/youtube/channel/123456/statistics", nil)

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

	if u != nil {
		defer resp.Body.Close()
		unmarshalBody(t, resp.Body, u)
	}

	return resp.StatusCode, resp.Status
}

func executePost(t *testing.T, url string, m json.Marshaler, u json.Unmarshaler) (int, string) {
	var reqBody io.Reader
	if m != nil {
		jsonData, err := m.MarshalJSON()
		require.NoError(t, err)
		reqBody = io.NopCloser(bytes.NewReader(jsonData))
	}

	resp, err := http.Post(url, "application/json", reqBody)
	require.NoError(t, err)

	if u != nil {
		defer resp.Body.Close()
		unmarshalBody(t, resp.Body, u)
	}

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

func createPgContainer(ctx context.Context) *helpers.PostgresContainer {
	pgContainer, err := helpers.CreatePostgresContainer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return pgContainer
}

func createApp(ctx context.Context, host string, port int, dbUrl string) *App {
	app, err := NewApp(ctx, host, port, dbUrl, "", mockClient)
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
