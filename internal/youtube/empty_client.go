package youtube

import "errors"

// ErrUnimplemented is returned by methods of EmptyClient to indicate no implementation exists.
var ErrUnimplemented = errors.New("unimplemented")

// EmptyClient is a placeholder implementation of the Client interface.
type EmptyClient struct{}

// GetChannelData is an unimplemented method of EmptyClient that returns an error.
func (e *EmptyClient) GetChannelData(channelID string) (string, error) {
	return "", ErrUnimplemented
}