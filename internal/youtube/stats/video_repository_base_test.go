//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"context"
	"io"
	"log/slog"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/suite"
)

type BaseVideoRepoTestSuite struct {
	suite.Suite
	repository internalVideoRepository
	ctx        context.Context
	conn       *pgx.Conn
	tx         pgx.Tx
}

func (suite *BaseVideoRepoTestSuite) SetupSuite() {
	ctx := context.Background()
	t := suite.T()
	suite.ctx = ctx

	conn, err := helpers.StartPgAndGetConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}

	suite.conn = conn
}

func (suite *BaseVideoRepoTestSuite) TearDownSuite() {
	if suite.conn != nil {
		_ = suite.conn.Close(suite.ctx)
	}
}

func (suite *BaseVideoRepoTestSuite) SetupTest() {
	t := suite.T()

	tx, err := suite.conn.Begin(suite.ctx)
	if err != nil {
		t.Fatal(err)
	}

	suite.tx = tx

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	suite.repository = &YoutubeVideoRepository{db: tx, logger: logger}
}

func (suite *BaseVideoRepoTestSuite) TearDownTest() {
	t := suite.T()

	if suite.tx != nil {
		if err := suite.tx.Rollback(suite.ctx); err != nil {
			t.Logf("Error rolling back transaction: %v", err)
		}
	}
}

func insertVideos(ctx context.Context, tx pgx.Tx, videos []YoutubeVideo) error {
	query := `
			INSERT INTO youtube_video (
					youtube_channel_id,
					external_id,
					title,
					published_at,
					created_at,
			        checked_at
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
			    $6
			)`

	for _, v := range videos {
		_, err := tx.Exec(ctx, query, v.YoutubeChannelId, v.ExternalID, v.Title, v.PublishedAt, v.CreatedAt, v.CheckedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func insertVideoStats(ctx context.Context, tx pgx.Tx, stats []YoutubeVideoStats) error {
	query := `
			INSERT INTO youtube_video_stat (
					youtube_video_id,
					view_count,
					like_count,
					comment_count,
					created_at
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5
			)`

	for _, s := range stats {
		_, err := tx.Exec(ctx, query, s.YoutubeVideoID, s.ViewCount, s.LikeCount, s.CommentCount, s.CreatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}
