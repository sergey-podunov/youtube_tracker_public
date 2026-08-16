//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"io"
	"log/slog"
	"testing"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYoutubeChannelRepository_CreateChannel(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, db helpers.Querier, repository *YoutubeChannelRepository)
	}

	tests := []testCase{
		{
			"empty db", func(t *testing.T, db helpers.Querier, repository *YoutubeChannelRepository) {
				channel, err := repository.createChannel(ctx,
					db,
					YoutubeChannel{
						ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Title:      "Google Developers",
					})
				require.NoError(t, err)
				require.NotNil(t, channel.YoutubeChannelId)

				var actualChannel YoutubeChannel
				err =
					db.QueryRow(ctx, "SELECT * FROM youtube_channel WHERE youtube_channel_id = $1", channel.YoutubeChannelId).
						Scan(&actualChannel.YoutubeChannelId,
							&actualChannel.ExternalID,
							&actualChannel.Title,
							&actualChannel.CustomURL,
							&actualChannel.Description,
							&actualChannel.PublishedAt,
							&actualChannel.CreatedAt,
							&actualChannel.CheckedAt)
				require.NoError(t, err)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
				assert.Equal(t, "Google Developers", actualChannel.Title)
				assert.False(t, actualChannel.CreatedAt.IsZero())
				assert.Equal(t, channel.CreatedAt, actualChannel.CreatedAt)
			},
		},
		{
			"channel exists", func(t *testing.T, db helpers.Querier, repository *YoutubeChannelRepository) {
				channel := YoutubeChannel{
					ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
					Title:      "Google Developers",
				}
				existingChannel, err := repository.createChannel(ctx, db, channel)
				require.NoError(t, err)

				newChannel, err := repository.createChannel(ctx, db, channel)
				assert.Nil(t, err)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", newChannel.ExternalID)
				assert.Equal(t, "Google Developers", newChannel.Title)
				assert.NotEqual(t, existingChannel.YoutubeChannelId, newChannel.YoutubeChannelId)
			},
		},
	}

	conn, err := helpers.StartPgAndGetConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		repository := &YoutubeChannelRepository{db: tx, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, tx, repository)
		})

		if err := tx.Rollback(ctx); err != nil {
			t.Logf("Error rolling back transaction test %s: %#v", tc.name, err)
		}
	}
}
