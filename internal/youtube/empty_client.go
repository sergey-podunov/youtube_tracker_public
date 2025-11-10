package youtube

import (
	"context"
	"errors"
)

var ErrUnimplemented = errors.New("unimplemented")

// EmptyClient is a placeholder implementation of the Client interface.
type EmptyClient struct{}

func (e *EmptyClient) GetChannelId(ctx context.Context, channelName string) (string, error) {
	return "", ErrUnimplemented
}

func (e *EmptyClient) GetChannelData(ctx context.Context, channelID string) (ChannelData, error) {
	return ChannelData{}, ErrUnimplemented
}
