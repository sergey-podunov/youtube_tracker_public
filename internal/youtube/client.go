package youtube

import (
	"context"
	"errors"
)

type Client interface {
	GetChannelData(ctx context.Context, channelId string) (*ChannelData, error)
}

type ChannelData struct {
	ChannelID        string `json:"channelId"`
	SubscribersCount int64  `json:"subscribersCount"`
}

type HttpClient struct {
	APIKey  string
	BaseURL string
}

// NewHttpClient initializes and returns a new instance of HttpClient.
func NewHttpClient(apiKey, baseURL string) *HttpClient {
	return &HttpClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// GetChannelData fetches and returns data for a specific YouTube channel.
// It uses the APIKey and BaseURL fields of the HttpClient struct.
func (c *HttpClient) GetChannelData(ctx context.Context, channelId string) (*ChannelData, error) {
	// This is a placeholder for actual implementation.
	// Replace it with logic to make an HTTP GET request to the YouTube API.
	return nil, errors.New("unimplemented")
}
