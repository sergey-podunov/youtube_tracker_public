package youtube

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	
	"youtube_tracker/internal/notify"
	
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Client interface {
	GetChannelId(ctx context.Context, channelName string) (string, error)
	GetChannelData(ctx context.Context, channelId string) (ChannelData, error)
}

type ChannelData struct {
	ChannelID        string `json:"channelId"`
	SubscribersCount int64  `json:"subscribersCount"`
}

type HttpClient struct {
	EmptyClient
	service  *youtube.Service
	notifier notify.Notifier
}

func NewHttpClient(ctx context.Context, notifier notify.Notifier, googleApiKey string) (*HttpClient, error) {
	service, err := youtube.NewService(ctx, option.WithAPIKey(googleApiKey))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Youtube service: %v", err)
	}

	return &HttpClient{
		service:  service,
		notifier: notifier,
	}, nil
}

func (c *HttpClient) GetChannelId(ctx context.Context, channelName string) (string, error) {
	call := c.service.Search.List([]string{"snippet"}).Q(channelName).Type("channel")

	response, err := call.Do()
	if err != nil {
		if isAuthError(err) {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API key issue during GetChannelId(%s): %v", channelName, err))
		} else {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API error in GetChannelId(%s): %v", channelName, err))
		}
		return "", fmt.Errorf("unable to retrieve channel id: %v", err)
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no channel found with the specified channel name: %s", channelName)
	}

	channelId := response.Items[0].Snippet.ChannelId

	return channelId, nil
}

func (c *HttpClient) GetChannelData(ctx context.Context, channelId string) (ChannelData, error) {
	call := c.service.Channels.List([]string{"snippet,contentDetails,statistics"})
	call = call.Id(channelId)

	response, err := call.Do()
	if err != nil {
		if isAuthError(err) {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API key issue during GetChannelData(%s): %v", channelId, err))
		} else {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API error in GetChannelData(%s): %v", channelId, err))
		}
		return ChannelData{}, fmt.Errorf("unable to retrieve channel data: %v", err)
	}

	if len(response.Items) == 0 {
		return ChannelData{}, fmt.Errorf("no channel found with the specified channel ID: %s", channelId)
	}

	log.Printf("Channel ID: %s\n", response.Items[0].Id)

	channel := response.Items[0]
	subscribersCount := channel.Statistics.SubscriberCount

	return ChannelData{
		ChannelID:        channelId,
		SubscribersCount: int64(subscribersCount),
	}, nil
}

// isAuthError detects common Google API errors indicating authentication/credential problems
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		if gErr.Code == 401 || gErr.Code == 403 {
			for _, e := range gErr.Errors {
				if e.Reason == "authError" || e.Reason == "invalid" || e.Reason == "forbidden" {
					return true
				}
			}
			m := strings.ToLower(gErr.Message)
			if strings.Contains(m, "auth") || strings.Contains(m, "api key") || strings.Contains(m, "credential") {
				return true
			}
		}
	}

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "api key") || strings.Contains(s, "invalid credentials")
}
