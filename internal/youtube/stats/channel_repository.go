package stats

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
)

type ChannelRepository interface {
	CreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, error)
	GetChannel(ctx context.Context, channelId int64) (*YoutubeChannel, error)
	GetChannels(ctx context.Context, sinceTime time.Time, count int) ([]YoutubeChannel, error)
	StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (*YoutubeChannelStats, error)
}

type YoutubeChannelRepository struct {
	querier helpers.Querier
}

func NewChannelRepository(ctx context.Context, connStr string) (*YoutubeChannelRepository, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	if err = conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	return &YoutubeChannelRepository{
		querier: conn,
	}, nil
}

func (r *YoutubeChannelRepository) CreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, error) {
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
	err := r.querier.QueryRow(ctx, query, channel.Name, channel.ExternalId, channel.CreatedAt).Scan(&insertedID)
	if err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return nil, err
	}

	channel.YoutubeChannelId = insertedID
	return &channel, nil
}

func (r *YoutubeChannelRepository) GetChannel(ctx context.Context, channelId int64) (*YoutubeChannel, error) {
	query := `
		SELECT 
			youtube_channel_id, 
			channel_name, 
			external_id, 
			created_at 
		FROM 
			youtube_channel 
		WHERE 
			youtube_channel_id = $1
	`
	var channel YoutubeChannel
	err := r.querier.QueryRow(ctx, query, channelId).
		Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalId, &channel.CreatedAt)
	if err != nil {
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
		RETURNING youtube_channel_stat_id
	`

	if stats.CreatedAt.IsZero() {
		stats.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	err := r.querier.QueryRow(ctx, query, stats.YoutubeChannelId, stats.SubscribersCount, stats.CreatedAt).Scan(&insertedID)
	if err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return nil, err
	}

	stats.YoutubeChannelStatId = insertedID
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
		err := rows.Scan(&channel.YoutubeChannelId, &channel.Name, &channel.ExternalId, &channel.CreatedAt, &channel.CheckedAt)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return channels, nil
}
