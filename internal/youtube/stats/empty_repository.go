package stats

import (
	"context"
	"fmt"
)

type EmptyChannelRepository struct{}

func (r *EmptyChannelRepository) CreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (r *EmptyChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (*YoutubeChannel, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (r *EmptyChannelRepository) StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (*YoutubeChannelStats, error) {
	return nil, fmt.Errorf("unimplemented")
}
