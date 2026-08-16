package stats

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
)

type VideoRepository interface {
	GetVideo(ctx context.Context, videoID int64) (YoutubeVideo, bool, error)
	GetVideos(ctx context.Context, sinceTime time.Time, count int) ([]YoutubeVideo, error)
	StoreVideoStats(ctx context.Context, stats YoutubeVideoStats) (YoutubeVideoStats, error)
}

type internalVideoRepository interface {
	VideoRepository
	createVideo(ctx context.Context, q helpers.Querier, video YoutubeVideo) (YoutubeVideo, error)
	getVideo(ctx context.Context, q helpers.Querier, videoID int64) (YoutubeVideo, bool, error)
	getVideoByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeVideo, bool, error)
	getVideoStat(ctx context.Context, q helpers.Querier, videoID int64) ([]YoutubeVideoStats, error)
	getVideosByChannelPaginated(ctx context.Context, q helpers.Querier, channelID int64, offset int, limit int) ([]YoutubeVideo, error)
	getVideosByChannelCount(ctx context.Context, q helpers.Querier, channelID int64) (int, error)
	updateVideoCheckedAt(ctx context.Context, q helpers.Querier, videoID int64, checkedAt time.Time) error
}

type YoutubeVideoRepository struct {
	db     Database
	logger *slog.Logger
}

const videoRepositoryComponentName = "YoutubeVideoRepository"

func NewVideoRepository(logger *slog.Logger, db Database) *YoutubeVideoRepository {
	return &YoutubeVideoRepository{
		db:     db,
		logger: logger.With(slog.String("component", videoRepositoryComponentName)),
	}
}

func (r *YoutubeVideoRepository) createVideo(ctx context.Context, q helpers.Querier, video YoutubeVideo) (YoutubeVideo, error) {
	logger := helpers.LoggerFromContextWithDefault(ctx, videoRepositoryComponentName, r.logger)
	logger.Info("Creating video", slog.Any("video", video))
	query := `
		INSERT INTO youtube_video (
			youtube_channel_id,
			external_id,
			title,
			published_at,
			created_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5
		)
		RETURNING youtube_video_id`

	if video.CreatedAt.IsZero() {
		video.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	if err := q.QueryRow(ctx, query, video.YoutubeChannelId, video.ExternalID, video.Title, video.PublishedAt, video.CreatedAt).Scan(&insertedID); err != nil {
		logger.Error("Failed to insert video", slog.Any("video", video), "error", err)
		return YoutubeVideo{}, err
	}

	video.YoutubeVideoId = insertedID

	return video, nil
}

func (r *YoutubeVideoRepository) GetVideo(ctx context.Context, videoID int64) (YoutubeVideo, bool, error) {
	return r.getVideo(ctx, r.db, videoID)
}

func (r *YoutubeVideoRepository) getVideo(ctx context.Context, q helpers.Querier, videoID int64) (YoutubeVideo, bool, error) {
	query := `
		SELECT
			youtube_video_id,
			youtube_channel_id,
			external_id,
			title,
			published_at,
			created_at,
			checked_at
		FROM
			youtube_video
		WHERE
			youtube_video_id = $1`

	var video YoutubeVideo
	err := q.QueryRow(ctx, query, videoID).
		Scan(&video.YoutubeVideoId, &video.YoutubeChannelId, &video.ExternalID, &video.Title, &video.PublishedAt, &video.CreatedAt, &video.CheckedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return YoutubeVideo{}, false, nil
		}
		return YoutubeVideo{}, false, err
	}

	return video, true, nil
}

func (r *YoutubeVideoRepository) getVideoByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeVideo, bool, error) {
	query := `
		SELECT
			youtube_video_id,
			youtube_channel_id,
			external_id,
			title,
			published_at,
			created_at,
			checked_at
		FROM
			youtube_video
		WHERE
			external_id = $1`

	var video YoutubeVideo
	err := q.QueryRow(ctx, query, externalID).
		Scan(&video.YoutubeVideoId, &video.YoutubeChannelId, &video.ExternalID, &video.Title, &video.PublishedAt, &video.CreatedAt, &video.CheckedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return YoutubeVideo{}, false, nil
		}
		return YoutubeVideo{}, false, err
	}

	return video, true, nil
}

func (r *YoutubeVideoRepository) StoreVideoStats(ctx context.Context, stats YoutubeVideoStats) (YoutubeVideoStats, error) {
	logger := helpers.LoggerFromContextWithDefault(ctx, videoRepositoryComponentName, r.logger)

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
		)
		RETURNING youtube_video_stat_id`

	if stats.CreatedAt.IsZero() {
		stats.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}

	var insertedID int64
	if err := r.db.QueryRow(ctx, query, stats.YoutubeVideoID, stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.CreatedAt).Scan(&insertedID); err != nil {
		logger.Error("Failed to insert video stat", slog.Any("video_stat", stats), "error", err)
		return YoutubeVideoStats{}, err
	}

	stats.YoutubeVideoStatID = insertedID

	return stats, nil
}

func (r *YoutubeVideoRepository) GetVideos(ctx context.Context, checkedBefore time.Time, count int) ([]YoutubeVideo, error) {
	query := `
		SELECT
			youtube_video_id,
			youtube_channel_id,
			external_id,
			title,
			published_at,
			created_at,
			checked_at
		FROM
			youtube_video
		WHERE (checked_at <= $1 or checked_at is null)
		LIMIT $2`
	rows, err := r.db.Query(ctx, query, checkedBefore, count)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var videos []YoutubeVideo

	for rows.Next() {
		var video YoutubeVideo
		if err := rows.Scan(&video.YoutubeVideoId, &video.YoutubeChannelId, &video.ExternalID, &video.Title, &video.PublishedAt, &video.CreatedAt, &video.CheckedAt); err != nil {
			return nil, err
		}

		videos = append(videos, video)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return videos, nil
}

func (r *YoutubeVideoRepository) getVideosByChannelPaginated(ctx context.Context, q helpers.Querier, channelID int64, offset int, limit int) ([]YoutubeVideo, error) {
	query := `
		SELECT
			youtube_video_id,
			youtube_channel_id,
			external_id,
			title,
			published_at,
			created_at,
			checked_at
		FROM
			youtube_video
		WHERE
			youtube_channel_id = $1
		ORDER BY published_at DESC
		OFFSET $2
		LIMIT $3`
	rows, err := q.Query(ctx, query, channelID, offset, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []YoutubeVideo
	for rows.Next() {
		var video YoutubeVideo
		if err := rows.Scan(&video.YoutubeVideoId, &video.YoutubeChannelId, &video.ExternalID, &video.Title, &video.PublishedAt, &video.CreatedAt, &video.CheckedAt); err != nil {
			return nil, err
		}

		videos = append(videos, video)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return videos, nil
}

func (r *YoutubeVideoRepository) getVideosByChannelCount(ctx context.Context, q helpers.Querier, channelID int64) (int, error) {
	query := `SELECT count(*) FROM youtube_video WHERE youtube_channel_id = $1`

	var count int
	err := q.QueryRow(ctx, query, channelID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *YoutubeVideoRepository) updateVideoCheckedAt(ctx context.Context, q helpers.Querier, videoID int64, checkedAt time.Time) error {
	query := `UPDATE youtube_video SET checked_at = $1 WHERE youtube_video_id = $2`
	_, err := q.Exec(ctx, query, checkedAt, videoID)
	return err
}

func (r *YoutubeVideoRepository) getVideoStat(ctx context.Context, q helpers.Querier, videoID int64) ([]YoutubeVideoStats, error) {
	query := `
		SELECT
			youtube_video_stat_id,
			youtube_video_id,
			view_count,
			like_count,
			comment_count,
			created_at
		FROM
			youtube_video_stat
		WHERE
			youtube_video_id = $1`

	rows, err := q.Query(ctx, query, videoID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	var stats []YoutubeVideoStats
	for rows.Next() {
		var stat YoutubeVideoStats
		if err := rows.Scan(&stat.YoutubeVideoStatID, &stat.YoutubeVideoID, &stat.ViewCount, &stat.LikeCount, &stat.CommentCount, &stat.CreatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
