package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Client interface {
	GetChannelId(channelName string) (string, error)
	GetChannelData(channelId string) (*ChannelData, error)
}

type ChannelData struct {
	ChannelID        string `json:"channelId"`
	SubscribersCount int64  `json:"subscribersCount"`
}

type HttpClient struct {
	EmptyClient
	service *youtube.Service
}

func NewHttpClient(ctx context.Context, authDirPath string) (*HttpClient, error) {
	clientSecretFilePath := filepath.Join(authDirPath, "client_secret.json")

	secretsConf, err := os.ReadFile(clientSecretFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(secretsConf, youtube.YoutubeReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client, err := getClient(ctx, authDirPath, config)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Youtube client: %v", err)
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Youtube service: %v", err)
	}

	return &HttpClient{
		service: service,
	}, nil
}

func (c *HttpClient) GetChannelId(channelName string) (string, error) {
	call := c.service.Search.List([]string{"snippet"}).Q(channelName).Type("channel")

	response, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve channel id: %v", err)
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no channel found with the specified channel name: %s", channelName)
	}

	channelId := response.Items[0].Snippet.ChannelId

	return channelId, nil
}

func (c *HttpClient) GetChannelData(channelId string) (*ChannelData, error) {
	call := c.service.Channels.List([]string{"snippet,contentDetails,statistics"})
	call = call.Id(channelId)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve channel data: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("no channel found with the specified channel ID: %s", channelId)
	}

	log.Printf("Channel ID: %s\n", response.Items[0].Id)

	channel := response.Items[0]
	subscribersCount := channel.Statistics.SubscriberCount

	return &ChannelData{
		ChannelID:        channelId,
		SubscribersCount: int64(subscribersCount),
	}, nil
}

func getClient(ctx context.Context, authDirPath string, config *oauth2.Config) (*http.Client, error) {
	cacheFile := tokenCacheFile(authDirPath)

	token, err := tokenFromFile(cacheFile)
	if err != nil {
		return nil, err
	}

	return config.Client(ctx, token), nil
}

func tokenCacheFile(authDirPath string) string {
	authCacheFilePath := filepath.Join(authDirPath, "auth-cache.json")
	return authCacheFilePath
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = f.Close()
	}()

	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)

	return t, err
}
