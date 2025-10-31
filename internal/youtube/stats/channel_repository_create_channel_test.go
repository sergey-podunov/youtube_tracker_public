//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
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
				channel, isNew, err := repository.getOrCreateChannel(ctx,
					db,
					YoutubeChannel{
						ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Name:       "Google Developers",
					})
				require.NoError(t, err)
				require.NotNil(t, channel.YoutubeChannelId)
				assert.True(t, isNew)

				var actualChannel YoutubeChannel
				err =
					db.QueryRow(ctx, "SELECT * FROM youtube_channel WHERE youtube_channel_id = $1", channel.YoutubeChannelId).
						Scan(&actualChannel.YoutubeChannelId, &actualChannel.ExternalID, &actualChannel.Name, &actualChannel.CreatedAt, &actualChannel.CheckedAt)
				require.NoError(t, err)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
				assert.Equal(t, "Google Developers", actualChannel.Name)
				assert.False(t, actualChannel.CreatedAt.IsZero())
				assert.Equal(t, channel.CreatedAt, actualChannel.CreatedAt)
			},
		},
		{
			"channel exists", func(t *testing.T, db helpers.Querier, repository *YoutubeChannelRepository) {
				channel := YoutubeChannel{
					ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
					Name:       "Google Developers",
				}
				_, _, err := repository.getOrCreateChannel(ctx, db, channel)
				require.NoError(t, err)

				existingChannel, isNew, err := repository.getOrCreateChannel(ctx, db, channel)
				assert.Nil(t, err)
				assert.False(t, isNew)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", existingChannel.ExternalID)
				assert.Equal(t, "Google Developers", existingChannel.Name)
				assert.NotNil(t, existingChannel.YoutubeChannelId)
				assert.False(t, existingChannel.CreatedAt.IsZero())
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
		repository := &YoutubeChannelRepository{querier: tx}

		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, tx, repository)
		})

		if err := tx.Rollback(ctx); err != nil {
			t.Logf("Error rolling back transaction test %s: %#v", tc.name, err)
		}
	}
}
