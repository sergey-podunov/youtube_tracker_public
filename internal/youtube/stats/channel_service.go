package stats

import (
	"context"
	"time"
	"youtube_tracker/internal/helpers"
	
	"github.com/jackc/pgx/v5"
)

type ChannelService interface {
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

func NewYoutubeChannelService(db helpers.TxController, repository internalChannelRepository) YoutubeChannelService {
	return YoutubeChannelService{
		db:         db,
		repository: repository,
	}
}

func (s YoutubeChannelService) GetChannelStats(ctx context.Context, ID int64, from *time.Time, to *time.Time) (YoutubeChannelStatsInfo, bool, error) {
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
			return  err
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
