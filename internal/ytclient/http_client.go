package ytclient

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

func (c *HttpClient) GetChannelData(ctx context.Context, channelId string) (ChannelData, error) {
	call := c.service.Channels.List([]string{"snippet,statistics,status"}).Id(channelId)
	return c.doChannelListCall(ctx, call, "GetChannelData", channelId)
}

func (c *HttpClient) GetChannelByHandle(ctx context.Context, handle string) (ChannelData, error) {
	call := c.service.Channels.List([]string{"snippet,statistics,status"}).ForHandle(handle)
	return c.doChannelListCall(ctx, call, "GetChannelByHandle", handle)
}

func (c *HttpClient) GetChannelByUsername(ctx context.Context, username string) (ChannelData, error) {
	call := c.service.Channels.List([]string{"snippet,statistics,status"}).ForUsername(username)
	return c.doChannelListCall(ctx, call, "GetChannelByUsername", username)
}

func (c *HttpClient) doChannelListCall(ctx context.Context, call *youtube.ChannelsListCall, method, lookup string) (ChannelData, error) {
	response, err := call.Do()
	if err != nil {
		if isAuthError(err) {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API key issue during %s(%s): %v", method, lookup, err))
		} else {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API error in %s(%s): %v", method, lookup, err))
		}
		return ChannelData{}, fmt.Errorf("unable to retrieve channel data: %v", err)
	}

	if len(response.Items) == 0 {
		return ChannelData{}, fmt.Errorf("no channel found for %s: %s", method, lookup)
	}

	log.Printf("Channel lookup %s(%s) resolved to ID: %s\n", method, lookup, response.Items[0].Id)

	channel := response.Items[0]
	data := ChannelData{ChannelID: channel.Id}

	if channel.Snippet != nil {
		data.Title = channel.Snippet.Title
		data.PublishedAt = channel.Snippet.PublishedAt
		data.Description = channel.Snippet.Description
		data.CustomURL = channel.Snippet.CustomUrl
		data.ThumbnailURL = getThumbnailURL(channel.Snippet.Thumbnails)
	}

	if channel.Statistics != nil {
		data.SubscribersCount = int64(channel.Statistics.SubscriberCount)
		data.ViewCount = int64(channel.Statistics.ViewCount)
		data.VideoCount = int64(channel.Statistics.VideoCount)
	}

	return data, nil
}

// GetChannelVideos returns a channel's most recent videos by reading its
// uploads playlist (playlistItems.list, 1 quota unit) instead of search.list
// (100 units). The API caps maxResults at 50, which fits in a single page, so
// no pagination is done.
func (c *HttpClient) GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]VideoData, error) {
	playlistID, err := UploadsPlaylistID(channelId)
	if err != nil {
		return nil, err
	}

	call := c.service.PlaylistItems.List([]string{"contentDetails"}).
		PlaylistId(playlistID).
		MaxResults(maxResults)

	response, err := call.Do()
	if err != nil {
		// The API answers 404 for a channel that has never uploaded (its
		// uploads playlist does not exist yet) — an empty result, not an error.
		if isNotFoundError(err) {
			return nil, nil
		}
		if isAuthError(err) {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API key issue during GetChannelVideos(%s): %v", channelId, err))
		} else {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API error in GetChannelVideos(%s): %v", channelId, err))
		}
		return nil, fmt.Errorf("unable to retrieve channel videos: %v", err)
	}

	videoIds := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if item.ContentDetails == nil {
			continue
		}
		videoIds = append(videoIds, item.ContentDetails.VideoId)
	}

	if len(videoIds) == 0 {
		return nil, nil
	}

	return c.GetVideosData(ctx, videoIds)
}

func (c *HttpClient) GetVideosData(ctx context.Context, videoIds []string) ([]VideoData, error) {
	call := c.service.Videos.List([]string{"snippet", "statistics"}).
		Id(strings.Join(videoIds, ","))

	response, err := call.Do()
	if err != nil {
		if isAuthError(err) {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API key issue during GetVideosData: %v", err))
		} else {
			_ = c.notifier.Send(ctx, fmt.Sprintf("YouTube API error in GetVideosData: %v", err))
		}
		return nil, fmt.Errorf("unable to retrieve videos data: %v", err)
	}

	videos := make([]VideoData, 0, len(response.Items))
	for _, item := range response.Items {
		videos = append(videos, VideoData{
			VideoID:      item.Id,
			ChannelID:    item.Snippet.ChannelId,
			Title:        item.Snippet.Title,
			PublishedAt:  item.Snippet.PublishedAt,
			ViewCount:    int64(item.Statistics.ViewCount),
			LikeCount:    int64(item.Statistics.LikeCount),
			CommentCount: int64(item.Statistics.CommentCount),
		})
	}

	return videos, nil
}

func isNotFoundError(err error) bool {
	var gErr *googleapi.Error
	return errors.As(err, &gErr) && gErr.Code == 404
}

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

func getThumbnailURL(thumbnails *youtube.ThumbnailDetails) string {
	if thumbnails == nil {
		return ""
	}
	if thumbnails.Default != nil && thumbnails.Default.Url != "" {
		return thumbnails.Default.Url
	}
	return ""
}
