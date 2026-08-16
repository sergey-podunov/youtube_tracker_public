//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"context"
	"testing"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

type channelStatView struct {
	YoutubeChannelID int64
	SubscribersCount int64
	ViewCount        int64
	VideoCount       int64
	CreatedAt        time.Time
}

func toYoutubeChannelStatsView(in []YoutubeChannelStats) []channelStatView {
	out := make([]channelStatView, len(in))
	for i, stat := range in {
		out[i] = channelStatView{
			YoutubeChannelID: stat.YoutubeChannelID,
			SubscribersCount: stat.SubscribersCount,
			VideoCount:       stat.VideoCount,
			ViewCount:        stat.ViewCount,
			CreatedAt:        stat.CreatedAt,
		}
	}

	return out
}

func TestYoutubeChannelRepository_GetChannelStat(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name            string
		statsBeforeTest []YoutubeChannelStats
		expectedStats   []channelStatView
	}

	tests := []testCase{
		{
			name:            "no stats in database",
			statsBeforeTest: []YoutubeChannelStats{},
			expectedStats:   []channelStatView{},
		},
		{
			name: "stats exists",
			statsBeforeTest: []YoutubeChannelStats{
				{
					YoutubeChannelID: 84374,
					SubscribersCount: 5,
					ViewCount:        6,
					VideoCount:       7,
					CreatedAt:        helpers.ParseTime("2025-11-12T05:06:07Z"),
				},
				{
					YoutubeChannelID: 84374,
					SubscribersCount: 7,
					ViewCount:        8,
					VideoCount:       9,
					CreatedAt:        helpers.ParseTime("2025-11-15T05:06:07Z"),
				},
				{
					YoutubeChannelID: 534,
					SubscribersCount: 15,
					CreatedAt:        helpers.ParseTime("2025-11-15T05:06:07Z"),
				},
			},
			expectedStats: []channelStatView{
				{
					YoutubeChannelID: 84374,
					SubscribersCount: 5,
					ViewCount:        6,
					VideoCount:       7,
					CreatedAt:        helpers.ParseTime("2025-11-12T05:06:07Z"),
				},
				{
					YoutubeChannelID: 84374,
					SubscribersCount: 7,
					ViewCount:        8,
					VideoCount:       9,
					CreatedAt:        helpers.ParseTime("2025-11-15T05:06:07Z"),
				},
			},
		},
	}

	conn, err := helpers.StartPgAndGetConnection(ctx)
	defer func() { _ = conn.Close(ctx) }()

	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := conn.Begin(ctx)
			defer func() { _ = tx.Rollback(ctx) }()

			if err != nil {
				t.Fatal(err)
			}

			channels := []YoutubeChannel{
				{
					YoutubeChannelId: int64(84374),
					ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
					Title:            "Google Developers",
				},
				{
					YoutubeChannelId: int64(534),
					ExternalID:       "abc",
					Title:            "another channel",
				},
			}
			if err := insertChannelsWithId(ctx, tx, channels); err != nil {
				t.Fatalf("Error inserting channels for subtest %s: %v", tc.name, err)
			}
			if err := insertStats(ctx, tx, tc.statsBeforeTest); err != nil {
				t.Fatalf("Error inserting channel stats for subtest %s: %#v", tc.name, err)
			}

			subtestRepo := &YoutubeChannelRepository{db: tx}
			actualStats, err := subtestRepo.getChannelStat(ctx, tx, int64(84374))
			if err != nil {
				t.Fatal(err)
			}

			assert.ElementsMatch(t, tc.expectedStats, toYoutubeChannelStatsView(actualStats), "Returned stats do not match expected (ignoring order)")
		})
	}
}

func insertStats(ctx context.Context, tx pgx.Tx, statsBeforeTest []YoutubeChannelStats) error {
	query := `
			INSERT INTO youtube_channel_stat (
					youtube_channel_id,
					created_at,
					subscribers_count,
			        video_count,
			        view_count
			) VALUES (
				$1, $2, $3, $4, $5
			)`

	for _, stat := range statsBeforeTest {
		_, err := tx.Exec(ctx, query,
			stat.YoutubeChannelID,
			stat.CreatedAt,
			stat.SubscribersCount,
			stat.VideoCount,
			stat.ViewCount)
		if err != nil {
			return err
		}
	}

	return nil
}
