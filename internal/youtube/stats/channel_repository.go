package stats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PoolMaxSize = 10
const PoolMinSize = 2
const PoolMaxConnectionIdleTime = 30 * time.Minute
const PoolMaxConnectionLifetime = time.Hour
const PoolHealthCheckPeriod = time.Minute

type ChannelRepository interface {
	GetOrCreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, bool, error)
	GetChannel(ctx context.Context, channelId int64) (*YoutubeChannel, error)
	GetChannels(ctx context.Context, sinceTime time.Time, count int) ([]YoutubeChannel, error)
	StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (*YoutubeChannelStats, error)
}

type YoutubeChannelRepository struct {
	querier helpers.Querier
	db      *pgxpool.Pool
}

func NewChannelRepository(ctx context.Context, connStr string) (*YoutubeChannelRepository, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to parse connection string: %v\n", err)
		return nil, err
	}

	config.MaxConns = PoolMaxSize
	config.MinConns = PoolMinSize
	config.MaxConnLifetime = PoolMaxConnectionLifetime
	config.MaxConnIdleTime = PoolMaxConnectionIdleTime
	config.HealthCheckPeriod = PoolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &YoutubeChannelRepository{
		querier: pool,
		db:      pool,
	}, nil
}

func (r *YoutubeChannelRepository) GetOrCreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, bool, error) {
	var channelFromRepo *YoutubeChannel
	var isNew bool
	err := helpers.RunInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		channel, newStatus, err := r.getOrCreateChannel(ctx, tx, channel)
		if err != nil {
			return err
		}
		channelFromRepo = channel
		isNew = newStatus
		return nil
	})

	return channelFromRepo, isNew, err
}

func (r *YoutubeChannelRepository) getOrCreateChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (*YoutubeChannel, bool, error) {
	oldChannel, err := r.getChannelByExternalId(ctx, q, channel.ExternalID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			newChannel, err := r.createChannel(ctx, q, channel)
			if err != nil {
				log.Printf("Failed to insert channel: %v\n", err)
				return nil, false, fmt.Errorf("failed to insert channel: %w", err)
			}

			return newChannel, true, nil
		}

		return nil, false, fmt.Errorf("failed to get channel by external ID: %w", err)
	}

	return oldChannel, false, nil
}

func (r *YoutubeChannelRepository) createChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (*YoutubeChannel, error) {
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
		RETURNING youtube_channel_id
	`

	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	if err := q.QueryRow(ctx, query, channel.Name, channel.ExternalID, channel.CreatedAt).Scan(&insertedID); err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return nil, err
	}

	channel.YoutubeChannelId = insertedID

	return &channel, nil
}

func (r *YoutubeChannelRepository) GetChannel(ctx context.Context, channelID int64) (*YoutubeChannel, error) {
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
	if err := r.querier.QueryRow(ctx, query, channelID).
		Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalID, &channel.CreatedAt); err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *YoutubeChannelRepository) getChannelByExternalId(ctx context.Context, q helpers.Querier, externalID string) (*YoutubeChannel, error) {
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
	if err := q.QueryRow(ctx, query, externalID).
		Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalID, &channel.CreatedAt); err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *YoutubeChannelRepository) StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (*YoutubeChannelStats, error) {
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
	if err := r.querier.QueryRow(ctx, query, stats.YoutubeChannelID, stats.SubscribersCount, stats.CreatedAt).Scan(&insertedID); err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return nil, err
	}

	stats.YoutubeChannelStatID = insertedID

	return &stats, nil
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
	rows, err := r.querier.Query(ctx, query, checkedBefore, count)

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
