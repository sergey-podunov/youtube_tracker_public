package ytclient

import (
	"context"
	"errors"
)

var ErrUnimplemented = errors.New("unimplemented")

// EmptyClient is a placeholder implementation of the Client interface.
type EmptyClient struct{}

func (e *EmptyClient) GetChannelData(ctx context.Context, channelID string) (ChannelData, error) {
	return ChannelData{}, ErrUnimplemented
}

func (e *EmptyClient) GetChannelByHandle(ctx context.Context, handle string) (ChannelData, error) {
	return ChannelData{}, ErrUnimplemented
}

func (e *EmptyClient) GetChannelByUsername(ctx context.Context, username string) (ChannelData, error) {
	return ChannelData{}, ErrUnimplemented
}

func (e *EmptyClient) GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]VideoData, error) {
	return nil, ErrUnimplemented
}

func (e *EmptyClient) GetVideosData(ctx context.Context, videoIds []string) ([]VideoData, error) {
	return nil, ErrUnimplemented
}
