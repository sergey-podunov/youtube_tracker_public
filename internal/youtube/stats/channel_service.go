package stats

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/storage"
	"youtube_tracker/internal/ytclient"

	"github.com/jackc/pgx/v5"
)

type ChannelService interface {
	CreateChannelFromURL(ctx context.Context, channelURL string) (YoutubeChannel, bool, error)
	GetChannelStats(ctx context.Context, ID int64, from *time.Time, to *time.Time) (YoutubeChannelStatsInfo, bool, error)
	GetChannels(ctx context.Context, page int, pageSize int) (YoutubeChannelsInfo, error)
}

type YoutubeChannelStatsInfo struct {
	ID         int64
	ChannelID  string
	Statistics []YoutubeChannelStats
}

const DefaultPageSize = 20

type YoutubeChannelsInfo struct {
	CurrentPage int
	TotalPages  int
	PageSize    int
	Channels    []YoutubeChannel
}

type YoutubeChannelService struct {
	db          helpers.TxController
	repository  internalChannelRepository
	client      ytclient.Client
	fileFetcher storage.FileFetcher
	logger      *slog.Logger
}

const channelServiceComponentName = "YoutubeChannelService"
const channelThumbnailKeyFormat = "/youtube/channel/%s.jpg"

func NewYoutubeChannelService(logger *slog.Logger, db helpers.TxController, repository internalChannelRepository, client ytclient.Client, fileFetcher storage.FileFetcher) *YoutubeChannelService {
	return &YoutubeChannelService{
		db:          db,
		repository:  repository,
		client:      client,
		fileFetcher: fileFetcher,
		logger:      logger.With(slog.String("component", channelServiceComponentName)),
	}
}

func (s *YoutubeChannelService) CreateChannelFromURL(ctx context.Context, channelURL string) (YoutubeChannel, bool, error) {
	parsed, err := ytclient.ParseChannelURL(channelURL)
	if err != nil {
		return YoutubeChannel{}, false, err
	}

	var data ytclient.ChannelData
	switch parsed.Type {
	case ytclient.LookupByID:
		data, err = s.client.GetChannelData(ctx, parsed.Value)
	case ytclient.LookupByHandle:
		data, err = s.client.GetChannelByHandle(ctx, parsed.Value)
	case ytclient.LookupByUsername:
		data, err = s.client.GetChannelByUsername(ctx, parsed.Value)
	}
	if err != nil {
		return YoutubeChannel{}, false, err
	}

	channel := YoutubeChannel{
		ExternalID:  data.ChannelID,
		Title:       data.Title,
		Description: &data.Description,
		CustomURL:   &data.CustomURL,
	}

	var createdChannel YoutubeChannel
	var isNew bool

	logger := helpers.LoggerFromContextWithDefault(ctx, channelServiceComponentName, s.logger)
	err = helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		existing, ok, txErr := s.repository.getChannelByExternalId(ctx, tx, channel.ExternalID)
		if txErr != nil {
			return txErr
		}

		if ok {
			logger.Warn("Channel already exists", slog.Any("channel", existing))
			createdChannel = existing
			return nil
		}

		newChannel, txErr := s.repository.createChannel(ctx, tx, channel)
		if txErr != nil {
			return txErr
		}

		createdChannel = newChannel
		isNew = true

		logger.Info("Channel created", slog.Any("channel", createdChannel))
		return nil
	})

	if err != nil {
		return createdChannel, isNew, err
	}

	if isNew && data.ThumbnailURL != "" && s.fileFetcher != nil {
		objectKey := fmt.Sprintf(channelThumbnailKeyFormat, data.ChannelID)
		if fetchErr := s.fileFetcher.FetchAndStore(ctx, data.ThumbnailURL, objectKey); fetchErr != nil {
			logger.Warn("Failed to fetch/store thumbnail", slog.String("channel_id", data.ChannelID), "error", fetchErr)
		}
	}

	return createdChannel, isNew, nil
}

func (s *YoutubeChannelService) GetChannelStats(ctx context.Context, ID int64, from *time.Time, to *time.Time) (YoutubeChannelStatsInfo, bool, error) {
	logger := helpers.LoggerFromContextWithDefault(ctx, channelServiceComponentName, s.logger)

	var statsInfo YoutubeChannelStatsInfo
	var ok bool
	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		channel, channelExists, err := s.repository.getChannel(ctx, tx, ID)
		if err != nil {
			return err
		}

		if !channelExists {
			logger.Info("Channel not found", slog.Int64("channel_id", ID))
			ok = false
			return nil
		}

		channelStats, err := s.repository.getChannelStat(ctx, tx, ID)
		if err != nil {
			return err
		}

		if len(channelStats) == 0 {
			logger.Info("There is no stat for channel", slog.Int64("channel_id", ID))
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

func (s *YoutubeChannelService) GetChannels(ctx context.Context, page int, pageSize int) (YoutubeChannelsInfo, error) {
	var youtubeChannelsInfo YoutubeChannelsInfo

	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		offset := getOffsetByPage(page, pageSize)
		pageSize = getPageSize(pageSize)

		channels, err := s.repository.getChannelsPaginated(ctx, tx, offset, pageSize)
		if err != nil {
			return err
		}

		channelsCount, err := s.repository.getChannelsCount(ctx, tx)
		if err != nil {
			return err
		}

		currentPage := getCurrentPage(page)
		totalPages := getTotalPages(channelsCount, pageSize)

		youtubeChannelsInfo = YoutubeChannelsInfo{
			CurrentPage: currentPage,
			TotalPages:  totalPages,
			PageSize:    pageSize,
			Channels:    channels,
		}
		return nil
	})

	return youtubeChannelsInfo, err
}

func getOffsetByPage(page int, pageSize int) int {
	var offset int
	if page > 0 {
		if page == 1 {
			offset = 0
		} else {
			offset = (page - 1) * pageSize
		}
	}
	return offset
}

func getPageSize(pageSize int) int {
	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > DefaultPageSize {
		pageSize = DefaultPageSize
	}
	return pageSize
}

func getCurrentPage(page int) int {
	var currentPage int
	if page == 0 {
		currentPage = 1
	} else {
		currentPage = page
	}
	return currentPage
}

func getTotalPages(channelsCount int, pageSize int) int {
	if channelsCount == 0 {
		return 0
	}
	return int(math.Ceil(float64(channelsCount) / float64(pageSize)))
}
