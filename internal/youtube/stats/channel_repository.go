package stats

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"
	
	"github.com/jackc/pgx/v5"
)

type ChannelRepository interface {
	GetChannel(ctx context.Context, channelID int64) (YoutubeChannel, bool, error)
	GetChannels(ctx context.Context, sinceTime time.Time, count int) ([]YoutubeChannel, error)
	StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (YoutubeChannelStats, error)
}

type internalChannelRepository interface {
	ChannelRepository
	createChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (YoutubeChannel, error)
	getChannel(ctx context.Context, q helpers.Querier, channelID int64) (YoutubeChannel, bool, error)
	getChannelStat(ctx context.Context, q helpers.Querier, channelID int64) ([]YoutubeChannelStats, error)
	getChannelByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeChannel, bool, error)
}

type Database interface {
	helpers.Querier
}

type YoutubeChannelRepository struct {
	db Database
	logger *slog.Logger
}

const channelRepositoryComponentName = "YoutubeChannelRepository"

func NewChannelRepository(logger *slog.Logger, db Database) *YoutubeChannelRepository {
	return &YoutubeChannelRepository{
		db: db,
		logger: logger.With(slog.String("component", channelRepositoryComponentName)),
	}
}

func (r *YoutubeChannelRepository) createChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (YoutubeChannel, error) {
	logger := helpers.LoggerFromContext(ctx, channelRepositoryComponentName, r.logger)
	logger.Info("Creating channel", slog.Any("channel", channel))
	query := `
		INSERT INTO youtube_channel (
			channel_name, 
			external_id, 
			created_at
		) VALUES (
			$1, 
			$2, 
			$3
		) 
		RETURNING youtube_channel_id`

	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	if err := q.QueryRow(ctx, query, channel.Name, channel.ExternalID, channel.CreatedAt).Scan(&insertedID); err != nil {
		logger.Error("Failed to insert channel", slog.Any("channel", channel), "error", err)
		return YoutubeChannel{}, err
	}

	channel.YoutubeChannelId = insertedID

	return channel, nil
}

func (r *YoutubeChannelRepository) GetChannel(ctx context.Context, channelID int64) (YoutubeChannel, bool, error) {
	return r.getChannel(ctx, r.db, channelID)
}

func (r *YoutubeChannelRepository) getChannel(ctx context.Context, q helpers.Querier, channelID int64) (YoutubeChannel, bool, error) {
	query := `
		SELECT
			youtube_channel_id,
			channel_name,
			external_id,
			created_at
		FROM
			youtube_channel
		WHERE
			youtube_channel_id = $1`

	var channel YoutubeChannel
	err := q.QueryRow(ctx, query, channelID).
		Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalID, &channel.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return YoutubeChannel{}, false, nil
		}
		return YoutubeChannel{}, false, err
	}

	return channel, true, nil
}

func (r *YoutubeChannelRepository) getChannelByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeChannel, bool, error) {
	query := `
		SELECT 
			youtube_channel_id, 
			channel_name, 
			external_id, 
			created_at 
		FROM 
			youtube_channel 
		WHERE 
			external_id = $1`

	var channel YoutubeChannel
	err := q.QueryRow(ctx, query, externalID).
		Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalID, &channel.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return YoutubeChannel{}, false, nil
		}
		return YoutubeChannel{}, false, err
	}

	return channel, true, nil
}

func (r *YoutubeChannelRepository) StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (YoutubeChannelStats, error) {
	logger := helpers.LoggerFromContext(ctx, channelRepositoryComponentName, r.logger)
	
	query := `
		INSERT INTO youtube_channel_stat (
			youtube_channel_id, 
			subscribers_count, 
			created_at
		) VALUES (
			$1, 
			$2, 
			$3
		) 
		RETURNING youtube_channel_stat_id`

	if stats.CreatedAt.IsZero() {
		stats.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	if err := r.db.QueryRow(ctx, query, stats.YoutubeChannelID, stats.SubscribersCount, stats.CreatedAt).Scan(&insertedID); err != nil {
		logger.Error("Failed to insert channel stat", slog.Any("channel_stat", stats), "error", err)
		return YoutubeChannelStats{}, err
	}

	stats.YoutubeChannelStatID = insertedID

	return stats, nil
}

func (r *YoutubeChannelRepository) GetChannels(ctx context.Context, checkedBefore time.Time, count int) ([]YoutubeChannel, error) {
	query := `
		SELECT 
			youtube_channel_id, 
			channel_name, 
			external_id, 
			created_at,
			checked_at
		FROM 
			youtube_channel
		WHERE (checked_at <= $1 or checked_at is null)
		limit $2`
	rows, err := r.db.Query(ctx, query, checkedBefore, count)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var channels []YoutubeChannel

	for rows.Next() {
		var channel YoutubeChannel
		if err := rows.Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalID, &channel.CreatedAt, &channel.CheckedAt); err != nil {
			return nil, err
		}

		channels = append(channels, channel)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return channels, nil
}

func (r *YoutubeChannelRepository) getChannelStat(ctx context.Context, q helpers.Querier, channelID int64) ([]YoutubeChannelStats, error) {
	query := `
		SELECT
			youtube_channel_stat_id,
			youtube_channel_id,
			subscribers_count,
			created_at
		FROM
			youtube_channel_stat
		WHERE
			youtube_channel_id = $1`

	rows, err := q.Query(ctx, query, channelID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	var stats []YoutubeChannelStats
	for rows.Next() {
		var stat YoutubeChannelStats
		if err := rows.Scan(&stat.YoutubeChannelStatID, &stat.YoutubeChannelID, &stat.SubscribersCount, &stat.CreatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
