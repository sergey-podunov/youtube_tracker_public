//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"testing"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type VideoRepoTestSuite struct {
	BaseVideoRepoTestSuite
}

func TestVideoRepoTestSuite(t *testing.T) {
	suite.Run(t, new(VideoRepoTestSuite))
}

func (suite *VideoRepoTestSuite) insertVideo(video YoutubeVideo) int64 {
	t := suite.T()
	ctx := suite.ctx

	query := `
	INSERT INTO youtube_video (
		youtube_channel_id,
		external_id,
		title,
		published_at,
		created_at,
		checked_at
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING youtube_video_id`

	var insertedID int64
	err := suite.tx.QueryRow(ctx, query, video.YoutubeChannelId, video.ExternalID, video.Title, video.PublishedAt, video.CreatedAt, video.CheckedAt).Scan(&insertedID)
	require.NoError(t, err)

	return insertedID
}

func (suite *VideoRepoTestSuite) TestCreateVideo() {
	t := suite.T()
	ctx := suite.ctx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	publishedAt := helpers.ParseTime("2025-03-04T05:06:07Z")
	video, err := suite.repository.createVideo(ctx, suite.tx, YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		PublishedAt:      &publishedAt,
	})
	require.NoError(t, err)
	require.NotZero(t, video.YoutubeVideoId)

	var actual YoutubeVideo
	err = suite.tx.QueryRow(ctx,
		`SELECT youtube_video_id, youtube_channel_id, external_id, title, published_at, created_at, checked_at
		 FROM youtube_video WHERE youtube_video_id = $1`, video.YoutubeVideoId).
		Scan(&actual.YoutubeVideoId, &actual.YoutubeChannelId, &actual.ExternalID, &actual.Title, &actual.PublishedAt, &actual.CreatedAt, &actual.CheckedAt)
	require.NoError(t, err)

	assert.Equal(t, channelID, actual.YoutubeChannelId)
	assert.Equal(t, "vid_ext_1", actual.ExternalID)
	assert.Equal(t, "Test Video", actual.Title)
	assert.Equal(t, publishedAt, *actual.PublishedAt)
	assert.False(t, actual.CreatedAt.IsZero())
	assert.Nil(t, actual.CheckedAt)
}

func (suite *VideoRepoTestSuite) TestGetVideo() {
	t := suite.T()
	ctx := suite.ctx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	publishedAt := helpers.ParseTime("2025-03-04T05:06:07Z")
	createdAt := helpers.ParseTime("2025-03-05T00:00:00Z")
	insertedID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		PublishedAt:      &publishedAt,
		CreatedAt:        createdAt,
	})

	actual, ok, err := suite.repository.GetVideo(ctx, insertedID)
	require.NoError(t, err)

	assert.True(t, ok)
	assert.Equal(t, insertedID, actual.YoutubeVideoId)
	assert.Equal(t, channelID, actual.YoutubeChannelId)
	assert.Equal(t, "vid_ext_1", actual.ExternalID)
	assert.Equal(t, "Test Video", actual.Title)
	assert.Equal(t, publishedAt, *actual.PublishedAt)
	assert.Equal(t, createdAt, actual.CreatedAt)
}

func (suite *VideoRepoTestSuite) TestGetVideoNotFound() {
	t := suite.T()
	ctx := suite.ctx

	actual, ok, err := suite.repository.GetVideo(ctx, 9999999)
	assert.NoError(t, err)
	assert.Equal(t, YoutubeVideo{}, actual)
	assert.False(t, ok)
}

func (suite *VideoRepoTestSuite) TestGetVideoByExternalId() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	createdAt := helpers.ParseTime("2025-03-05T00:00:00Z")
	insertedID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		CreatedAt:        createdAt,
	})

	actual, ok, err := suite.repository.getVideoByExternalId(ctx, tx, "vid_ext_1")
	require.NoError(t, err)

	assert.True(t, ok)
	assert.Equal(t, insertedID, actual.YoutubeVideoId)
	assert.Equal(t, "vid_ext_1", actual.ExternalID)
}

func (suite *VideoRepoTestSuite) TestGetVideoByExternalIdNotFound() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	actual, ok, err := suite.repository.getVideoByExternalId(ctx, tx, "missing_ext_id")
	assert.NoError(t, err)
	assert.Equal(t, YoutubeVideo{}, actual)
	assert.False(t, ok)
}

func (suite *VideoRepoTestSuite) TestStoreVideoStats() {
	t := suite.T()
	ctx := suite.ctx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	videoID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		CreatedAt:        helpers.ParseTime("2025-03-05T00:00:00Z"),
	})

	stored, err := suite.repository.StoreVideoStats(ctx, YoutubeVideoStats{
		YoutubeVideoID: videoID,
		ViewCount:      1000,
		LikeCount:      100,
		CommentCount:   10,
	})
	require.NoError(t, err)
	require.NotZero(t, stored.YoutubeVideoStatID)

	var actual YoutubeVideoStats
	err = suite.tx.QueryRow(ctx,
		`SELECT youtube_video_stat_id, youtube_video_id, view_count, like_count, comment_count, created_at
		 FROM youtube_video_stat WHERE youtube_video_stat_id = $1`, stored.YoutubeVideoStatID).
		Scan(&actual.YoutubeVideoStatID, &actual.YoutubeVideoID, &actual.ViewCount, &actual.LikeCount, &actual.CommentCount, &actual.CreatedAt)
	require.NoError(t, err)

	assert.Equal(t, videoID, actual.YoutubeVideoID)
	assert.Equal(t, int64(1000), actual.ViewCount)
	assert.Equal(t, int64(100), actual.LikeCount)
	assert.Equal(t, int64(10), actual.CommentCount)
	assert.False(t, actual.CreatedAt.IsZero())
}

func (suite *VideoRepoTestSuite) TestGetVideoStat() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	videoID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		CreatedAt:        helpers.ParseTime("2025-03-05T00:00:00Z"),
	})

	otherVideoID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_2",
		Title:            "Other Video",
		CreatedAt:        helpers.ParseTime("2025-03-05T00:00:00Z"),
	})

	statsToInsert := []YoutubeVideoStats{
		{YoutubeVideoID: videoID, ViewCount: 100, LikeCount: 10, CommentCount: 1, CreatedAt: helpers.ParseTime("2025-04-01T00:00:00Z")},
		{YoutubeVideoID: videoID, ViewCount: 200, LikeCount: 20, CommentCount: 2, CreatedAt: helpers.ParseTime("2025-04-02T00:00:00Z")},
		{YoutubeVideoID: otherVideoID, ViewCount: 999, LikeCount: 99, CommentCount: 9, CreatedAt: helpers.ParseTime("2025-04-02T00:00:00Z")},
	}
	require.NoError(t, insertVideoStats(ctx, tx, statsToInsert))

	actual, err := suite.repository.getVideoStat(ctx, tx, videoID)
	require.NoError(t, err)

	assert.Len(t, actual, 2)
	for _, s := range actual {
		assert.Equal(t, videoID, s.YoutubeVideoID)
	}
}

func (suite *VideoRepoTestSuite) TestGetVideos() {
	ctx := suite.ctx

	type testCase struct {
		name             string
		videosBeforeTest []YoutubeVideo
		count            int
		checkedBefore    time.Time
		expectedIds      []string
	}

	createdAt := helpers.ParseTime("2025-01-01T00:00:00Z")
	tests := []testCase{
		{
			name:             "No videos in database",
			videosBeforeTest: []YoutubeVideo{},
			count:            10,
			checkedBefore:    helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:      []string{},
		},
		{
			name: "checkedAt is null",
			videosBeforeTest: []YoutubeVideo{
				{ExternalID: "v1", Title: "Video 1", CreatedAt: createdAt},
			},
			count:         10,
			checkedBefore: helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:   []string{"v1"},
		},
		{
			name: "matched by checkedAt",
			videosBeforeTest: []YoutubeVideo{
				{ExternalID: "v2", Title: "Video 2", CreatedAt: createdAt, CheckedAt: helpers.Ptr(helpers.ParseTime("2025-10-01T00:00:00Z"))},
				{ExternalID: "v3", Title: "Video 3", CreatedAt: createdAt, CheckedAt: helpers.Ptr(helpers.ParseTime("2025-09-01T00:00:00Z"))},
			},
			count:         10,
			checkedBefore: helpers.ParseTime("2025-09-24T00:00:00Z"),
			expectedIds:   []string{"v3"},
		},
		{
			name: "limit by count",
			videosBeforeTest: []YoutubeVideo{
				{ExternalID: "v4", Title: "Video 4", CreatedAt: createdAt},
				{ExternalID: "v5", Title: "Video 5", CreatedAt: createdAt},
				{ExternalID: "v6", Title: "Video 6", CreatedAt: createdAt},
			},
			count:         2,
			checkedBefore: helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:   []string{"v4", "v5"},
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			t := suite.T()

			tx, err := suite.conn.Begin(ctx)
			require.NoError(t, err)
			defer func() {
				if err := tx.Rollback(ctx); err != nil {
					t.Logf("Error rolling back transaction for subtest %s: %v", tc.name, err)
				}
			}()

			channelID, err := insertChannel(ctx, tx, YoutubeChannel{
				ExternalID: "UC-channel",
				Title:      "Channel",
				CreatedAt:  createdAt,
			})
			require.NoError(t, err)

			for i := range tc.videosBeforeTest {
				tc.videosBeforeTest[i].YoutubeChannelId = channelID
			}
			if err := insertVideos(ctx, tx, tc.videosBeforeTest); err != nil {
				t.Fatalf("Error inserting videos for subtest %s: %v", tc.name, err)
			}

			subtestRepo := &YoutubeVideoRepository{db: tx}
			actualVideos, err := subtestRepo.GetVideos(ctx, tc.checkedBefore, tc.count)
			require.NoError(t, err)

			var actualExternalIds []string
			for _, v := range actualVideos {
				actualExternalIds = append(actualExternalIds, v.ExternalID)
			}
			assert.ElementsMatch(suite.T(), tc.expectedIds, actualExternalIds, "Returned videos do not match expected IDs (ignoring order)")
		})
	}
}

func (suite *VideoRepoTestSuite) TestGetVideosByChannelPaginated() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	earliest := helpers.ParseTime("2025-01-01T00:00:00Z")
	middle := helpers.ParseTime("2025-02-01T00:00:00Z")
	latest := helpers.ParseTime("2025-03-01T00:00:00Z")
	createdAt := helpers.ParseTime("2025-04-01T00:00:00Z")

	suite.insertVideo(YoutubeVideo{YoutubeChannelId: channelID, ExternalID: "v_earliest", Title: "earliest", PublishedAt: &earliest, CreatedAt: createdAt})
	middleID := suite.insertVideo(YoutubeVideo{YoutubeChannelId: channelID, ExternalID: "v_middle", Title: "middle", PublishedAt: &middle, CreatedAt: createdAt})
	suite.insertVideo(YoutubeVideo{YoutubeChannelId: channelID, ExternalID: "v_latest", Title: "latest", PublishedAt: &latest, CreatedAt: createdAt})

	actual, err := suite.repository.getVideosByChannelPaginated(ctx, tx, channelID, 1, 1)
	require.NoError(t, err)

	require.Len(t, actual, 1)
	assert.Equal(t, middleID, actual[0].YoutubeVideoId)
	assert.Equal(t, "v_middle", actual[0].ExternalID)
}

func (suite *VideoRepoTestSuite) TestGetVideosByChannelPaginatedEmpty() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	actual, err := suite.repository.getVideosByChannelPaginated(ctx, tx, 0, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, actual)
}

func (suite *VideoRepoTestSuite) TestGetVideosByChannelCount() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	otherChannelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-2",
		Title:      "Channel 2",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	createdAt := helpers.ParseTime("2025-04-01T00:00:00Z")
	suite.insertVideo(YoutubeVideo{YoutubeChannelId: channelID, ExternalID: "v1", Title: "v1", CreatedAt: createdAt})
	suite.insertVideo(YoutubeVideo{YoutubeChannelId: channelID, ExternalID: "v2", Title: "v2", CreatedAt: createdAt})
	suite.insertVideo(YoutubeVideo{YoutubeChannelId: otherChannelID, ExternalID: "v3", Title: "v3", CreatedAt: createdAt})

	count, err := suite.repository.getVideosByChannelCount(ctx, tx, channelID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func (suite *VideoRepoTestSuite) TestUpdateVideoCheckedAt() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	channelID, err := insertChannel(ctx, suite.tx, YoutubeChannel{
		ExternalID: "UC-channel-1",
		Title:      "Channel 1",
		CreatedAt:  helpers.ParseTime("2025-01-01T00:00:00Z"),
	})
	require.NoError(t, err)

	videoID := suite.insertVideo(YoutubeVideo{
		YoutubeChannelId: channelID,
		ExternalID:       "vid_ext_1",
		Title:            "Test Video",
		CreatedAt:        helpers.ParseTime("2025-03-05T00:00:00Z"),
	})

	checkedAt := helpers.ParseTime("2025-05-01T12:34:56Z")
	err = suite.repository.updateVideoCheckedAt(ctx, tx, videoID, checkedAt)
	require.NoError(t, err)

	var actual *time.Time
	err = suite.tx.QueryRow(ctx, `SELECT checked_at FROM youtube_video WHERE youtube_video_id = $1`, videoID).Scan(&actual)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, checkedAt, *actual)
}
