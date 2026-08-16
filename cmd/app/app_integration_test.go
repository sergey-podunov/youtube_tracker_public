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
	"youtube_tracker/internal/notify"
	"youtube_tracker/internal/ytclient"

	"github.com/ogen-go/ogen/conv"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

const host = "localhost"
const port = 8081

type MockYoutubeClient struct {
	GetChannelDataFunc       func(channelId string) (ytclient.ChannelData, error)
	GetChannelByHandleFunc   func(handle string) (ytclient.ChannelData, error)
	GetChannelByUsernameFunc func(username string) (ytclient.ChannelData, error)
	GetChannelVideosFunc     func(channelId string, maxResults int64) ([]ytclient.VideoData, error)
	GetVideosDataFunc        func(videoIds []string) ([]ytclient.VideoData, error)
}

func (m *MockYoutubeClient) GetChannelData(ctx context.Context, channelId string) (ytclient.ChannelData, error) {
	if m.GetChannelDataFunc != nil {
		return m.GetChannelDataFunc(channelId)
	}
	return ytclient.ChannelData{}, fmt.Errorf("GetChannelData not implemented in mock")
}

func (m *MockYoutubeClient) GetChannelByHandle(ctx context.Context, handle string) (ytclient.ChannelData, error) {
	if m.GetChannelByHandleFunc != nil {
		return m.GetChannelByHandleFunc(handle)
	}
	return ytclient.ChannelData{}, fmt.Errorf("GetChannelByHandle not implemented in mock")
}

func (m *MockYoutubeClient) GetChannelByUsername(ctx context.Context, username string) (ytclient.ChannelData, error) {
	if m.GetChannelByUsernameFunc != nil {
		return m.GetChannelByUsernameFunc(username)
	}
	return ytclient.ChannelData{}, fmt.Errorf("GetChannelByUsername not implemented in mock")
}

func (m *MockYoutubeClient) GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]ytclient.VideoData, error) {
	if m.GetChannelVideosFunc != nil {
		return m.GetChannelVideosFunc(channelId, maxResults)
	}
	return nil, fmt.Errorf("GetChannelVideos not implemented in mock")
}

func (m *MockYoutubeClient) GetVideosData(ctx context.Context, videoIds []string) ([]ytclient.VideoData, error) {
	if m.GetVideosDataFunc != nil {
		return m.GetVideosDataFunc(videoIds)
	}
	return nil, fmt.Errorf("GetVideosData not implemented in mock")
}

var mockClient = &MockYoutubeClient{
	GetChannelDataFunc: func(channelId string) (ytclient.ChannelData, error) {
		switch channelId {
		case "TestYoutubeID":
			return ytclient.ChannelData{
				ChannelID:        "TestYoutubeID",
				Title:            "Test Channel",
				Description:      "This is a test channel",
				CustomURL:        "@TestChannel",
				SubscribersCount: 151617,
				ViewCount:        123456789,
				VideoCount:       34562,
			}, nil
		case "TestYoutubeListID":
			return ytclient.ChannelData{
				ChannelID: "TestYoutubeListID",
				Title:     "Test Channel for list",
			}, nil
		case "TestYoutubeID_conflict":
			return ytclient.ChannelData{
				ChannelID: "TestYoutubeID_conflict",
				Title:     "Test Channel conflict",
			}, nil
		}
		return ytclient.ChannelData{}, fmt.Errorf("channel data not found for ID: %s", channelId)
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
			&api.YoutubeChannelRequest{URL: "https://youtube.com/channel/TestYoutubeID"},
			channelPostResp,
		)

		assert.Equal(t, http.StatusCreated, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel", channelPostResp.Title)
		assert.Equal(t, "TestYoutubeID", channelPostResp.YoutubeID)
		assert.NotEmpty(t, channelPostResp.CreatedAt.Value)
		assert.Greater(t, channelPostResp.ID.Value, int64(0))

		channelGetResp := api.YoutubeChannel{}
		respCode, respStatus = executeGet(t, url+"/youtube/channel/"+conv.Int64ToString(channelPostResp.ID.Value), &channelGetResp)

		assert.Equal(t, http.StatusOK, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel", channelGetResp.Title)
		assert.Equal(t, "This is a test channel", channelGetResp.Description.Value)
		assert.Equal(t, "@TestChannel", channelGetResp.CustomURL.Value)
		assert.Equal(t, "TestYoutubeID", channelGetResp.YoutubeID)
		assert.Equal(t, channelPostResp.CreatedAt.Value, channelGetResp.CreatedAt.Value)
		assert.Equal(t, channelPostResp.ID.Value, channelGetResp.ID.Value)
	})

	t.Run("Test /youtube/channels without params", func(t *testing.T) {
		channelPostResp := &api.YoutubeChannelClientError{}
		respCode, respStatus := executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannelRequest{},
			channelPostResp,
		)

		assert.Equal(t, http.StatusBadRequest, respCode, "Response code: %s", respStatus)
		assert.Truef(t, len(channelPostResp.Message) != 0, "Channel post response message should not be empty")
	})

	t.Run("Test /youtube/channels with wrong url", func(t *testing.T) {
		channelPostResp := &api.YoutubeChannelClientError{}
		respCode, respStatus := executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannelRequest{URL: "https://youtube.com/"},
			channelPostResp,
		)

		assert.Equal(t, http.StatusBadRequest, respCode, "Response code: %s", respStatus)
		assert.Truef(t, len(channelPostResp.Message) != 0, "Channel post response message should not be empty")
	})

	t.Run("/youtube/channel posts conflict", func(t *testing.T) {
		executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannelRequest{URL: "https://youtube.com/channel/TestYoutubeID_conflict"},
			nil,
		)

		channelErrorResp := &api.YoutubeChannelPostConflict{}
		respCode, respStatus := executePost(
			t,
			url+"/youtube/channel",
			&api.YoutubeChannelRequest{URL: "https://youtube.com/channel/TestYoutubeID_conflict"},
			channelErrorResp,
		)

		assert.Equal(t, http.StatusConflict, respCode, "Response code: %s", respStatus)
		assert.Equal(t, "Test Channel conflict", channelErrorResp.Title)
		assert.Equal(t, "TestYoutubeID_conflict", channelErrorResp.YoutubeID)
	})

	t.Run("Test /youtube/channel not found", func(t *testing.T) {
		channelGetResp := api.YoutubeChannel{}
		respCode, respStatus := executeGet(t, url+"/youtube/channel/123456", &channelGetResp)

		assert.Equal(t, http.StatusNotFound, respCode, "Response code: %s", respStatus)
	})

	t.Run("Test YouTube channel statistics", func(t *testing.T) {
		channelPostResp := &api.YoutubeChannel{}
		_, _ = executePost(t, url+"/youtube/channel", &api.YoutubeChannelRequest{URL: "https://youtube.com/channel/TestYoutubeID"}, channelPostResp)

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
	app, err := NewApp(ctx, port, dbUrl, "", "*", notify.NotifierConfig{}, nil, mockClient)
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
