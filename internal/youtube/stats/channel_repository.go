package stats

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

type ChannelRepository struct {
	conn *pgx.Conn
}

func NewChannelRepository(ctx context.Context, connStr string) (*ChannelRepository, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}
	return &ChannelRepository{
		conn: conn,
	}, nil
}

func (r *ChannelRepository) createChannel(ctx context.Context, channel YoutubeChannel) (YoutubeChannel, error) {
	query := `INSERT INTO youtube_channel(channel_name, external_id, created_at) VALUES ($1, $2, $3) RETURNING youtube_channel_id`

	var insertedID int
	err := r.conn.QueryRow(ctx, query, channel.name, channel.externalId, channel.createdAt).Scan(&insertedID)
	if err != nil {
		log.Printf("Failed to insert channel: %v\n", err)
		return YoutubeChannel{}, err
	}

	channel.youtubeChannelId = insertedID
	return channel, nil
}

/*
func (r *ChannelRepository) GetAllUsers(ctx context.Context) ([]YoutubeChannel, error) {
	query := `SELECT id, name, email FROM users`
	rows, err := r.conn.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to query users: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var users []YoutubeChannel

	for rows.Next() {
		var channel YoutubeChannel
		if err := rows.Scan(&channel.ID, &channel.Name, &channel.Email); err != nil {
			log.Printf("Failed to scan channel row: %v\n", err)
			continue
		}
		users = append(users, channel)
	}

	if rows.Err() != nil {
		log.Printf("Error while reading rows: %v\n", rows.Err())
		return nil, rows.Err()
	}

	return users, nil
}*/
