//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"context"
	"youtube_tracker/internal/helpers"
)

func insertChannel(ctx context.Context, querier helpers.Querier, channel YoutubeChannel) (int64, error) {
	query := `
	INSERT INTO youtube_channel (
		title,
		external_id,
		created_at
	)
	VALUES (
		$1,
		$2,
		$3
	) RETURNING youtube_channel_id`

	var insertedID int64
	err := querier.QueryRow(ctx, query, channel.Title, channel.ExternalID, channel.CreatedAt).Scan(&insertedID)
	if err != nil {
		return 0, err
	}

	return insertedID, nil
}
