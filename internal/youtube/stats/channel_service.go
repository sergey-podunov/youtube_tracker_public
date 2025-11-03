package stats

import (
	"context"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
)

type ChannelService interface {
	CreateChannel(ctx context.Context, channel YoutubeChannel) (YoutubeChannel, bool, error)
	GetChannelStats(ctx context.Context, ID int64, from *time.Time, to *time.Time) (YoutubeChannelStatsInfo, bool, error)
}

type YoutubeChannelStatsInfo struct {
	ID         int64
	ChannelID  string
	Statistics []YoutubeChannelStats
}

type YoutubeChannelService struct {
	db         helpers.TxController
	repository internalChannelRepository
}

func NewYoutubeChannelService(db helpers.TxController, repository internalChannelRepository) *YoutubeChannelService {
	return &YoutubeChannelService{
		db:         db,
		repository: repository,
	}
}

func (s *YoutubeChannelService) GetChannelStats(ctx context.Context, ID int64, from *time.Time, to *time.Time) (YoutubeChannelStatsInfo, bool, error) {
	var statsInfo YoutubeChannelStatsInfo
	var ok bool
	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		channel, channelExists, err := s.repository.getChannel(ctx, tx, ID)
		if err != nil {
			return err
		}

		if !channelExists {
			ok = false
			return nil
		}

		channelStats, err := s.repository.getChannelStat(ctx, tx, ID)
		if err != nil {
			return err
		}

		for i := range channelStats {
			channelStats[i].CreatedAt = channelStats[i].CreatedAt.Truncate(time.Hour * 24)
		}

		statsInfo = YoutubeChannelStatsInfo{
			ID:         channel.YoutubeChannelId,
			ChannelID:  channel.ExternalID,
			Statistics: channelStats,
		}
		ok = true

		return nil
	})

	return statsInfo, ok, err
}

func (s *YoutubeChannelService) CreateChannel(ctx context.Context, channel YoutubeChannel) (YoutubeChannel, bool, error) {
	var createdYoutubeChannel YoutubeChannel
	var channelCreated bool

	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		existingChannel, ok, err := s.repository.getChannelByExternalId(ctx, tx, channel.ExternalID)
		if err != nil {
			return err
		}

		if ok {
			createdYoutubeChannel = existingChannel
			return nil
		}

		newChannel, err := s.repository.createChannel(ctx, tx, channel)
		if err != nil {
			return err
		}

		createdYoutubeChannel = newChannel
		channelCreated = true

		return nil
	})

	return createdYoutubeChannel, channelCreated, err
}
