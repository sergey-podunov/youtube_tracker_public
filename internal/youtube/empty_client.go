package youtube

import "errors"

var ErrUnimplemented = errors.New("unimplemented")

// EmptyClient is a placeholder implementation of the Client interface.
type EmptyClient struct{}

func (e *EmptyClient) GetChannelId(channelName string) (string, error) {
	return "", ErrUnimplemented
}

func (e *EmptyClient) GetChannelData(channelID string) (*ChannelData, error) {
	return nil, ErrUnimplemented
}
