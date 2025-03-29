package stats

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

type ChannelRepository interface {
	CreateChannel(ctx context.Context, channel YoutubeChannel) (*YoutubeChannel, error)
	GetChannel(ctx context.Context, channelId int64) (*YoutubeChannel, error)
	StoreSubscriptionsCount(ctx context.Context, stats YoutubeChannelStats) (*YoutubeChannelStats, error)
}

type YoutubeChannelRepository struct {
	conn *pgx.Conn
}

func NewChannelRepository(ctx context.Context, connStr string) (*YoutubeChannelRepository, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}
	return &YoutubeChannelRepository{
		conn: conn,
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
	err := r.conn.QueryRow(ctx, query, channel.Name, channel.ExternalId, channel.CreatedAt).Scan(&insertedID)
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
	err := r.conn.QueryRow(ctx, query, channelId).
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
	err := r.conn.QueryRow(ctx, query, stats.YoutubeChannelId, stats.SubscribersCount, stats.CreatedAt).Scan(&insertedID)
	if err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return nil, err
	}

	stats.YoutubeChannelStatId = insertedID
	return &stats, nil
}
