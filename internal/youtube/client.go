package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"youtube_tracker/internal/notify"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

func NewHttpClient(ctx context.Context, notifier notify.Notifier, authDirPath string) (*HttpClient, error) {
	clientSecretFilePath := filepath.Join(authDirPath, "client_secret.json")

	secretsConf, err := os.ReadFile(clientSecretFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(secretsConf, youtube.YoutubeReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client, err := createClient(ctx, authDirPath, config)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Youtube client: %v", err)
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
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
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube OAuth token issue during GetChannelId(%s): %v", channelName, err))
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
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube OAuth token issue during GetChannelData(%s): %v", channelId, err))
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

func createClient(ctx context.Context, authDirPath string, config *oauth2.Config) (*http.Client, error) {
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

// isAuthError detects common Google API/OAuth errors indicating token/credential problems
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
			if strings.Contains(m, "auth") || strings.Contains(m, "token") || strings.Contains(m, "credential") {
				return true
			}
		}
	}
	var rErr *oauth2.RetrieveError
	if errors.As(err, &rErr) {
		body := strings.ToLower(string(rErr.Body))
		if strings.Contains(body, "invalid_grant") || strings.Contains(body, "expired") || strings.Contains(body, "revoked") {
			return true
		}
	}
	
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "invalid_grant") || strings.Contains(s, "token has been expired")
}
