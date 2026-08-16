package stats

import (
	"context"
	"errors"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/ytclient"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

var errUnimplemented = errors.New("unimplemented")

type MockYoutubeClient struct {
	mock.Mock
}

func (m *MockYoutubeClient) GetChannelData(ctx context.Context, channelId string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, channelId)
	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}
	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelByHandle(ctx context.Context, handle string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, handle)
	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}
	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelByUsername(ctx context.Context, username string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, username)
	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}
	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]ytclient.VideoData, error) {
	args := m.Called(ctx, channelId, maxResults)
	var data []ytclient.VideoData
	if args.Get(0) != nil {
		data = args.Get(0).([]ytclient.VideoData)
	}
	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetVideosData(ctx context.Context, videoIds []string) ([]ytclient.VideoData, error) {
	args := m.Called(ctx, videoIds)
	var data []ytclient.VideoData
	if args.Get(0) != nil {
		data = args.Get(0).([]ytclient.VideoData)
	}
	return data, args.Error(1)
}

type MockChannelRepository struct {
	mock.Mock
}

func (r *MockChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (YoutubeChannel, bool, error) {
	args := r.Called(ctx, youtubeChannelId)

	var ch YoutubeChannel
	if args.Get(0) != nil {
		ch = args.Get(0).(YoutubeChannel)
	}

	return ch, args.Get(1).(bool), args.Error(2)
}

func (r *MockChannelRepository) StoreSubscriptionsCount(ctx context.Context, channelStats YoutubeChannelStats) (YoutubeChannelStats, error) {
	args := r.Called(ctx, channelStats)

	var cs YoutubeChannelStats
	if args.Get(0) != nil {
		cs = args.Get(0).(YoutubeChannelStats)
	}

	return cs, args.Error(1)
}

func (r *MockChannelRepository) GetChannels(ctx context.Context, checkedBefore time.Time, count int) ([]YoutubeChannel, error) {
	args := r.Called(ctx, checkedBefore, count)

	var channels []YoutubeChannel
	if args.Get(0) != nil {
		channels = args.Get(0).([]YoutubeChannel)
	}

	return channels, args.Error(1)
}

type MockVideoRepository struct {
	mock.Mock
}

func (r *MockVideoRepository) GetVideo(ctx context.Context, videoID int64) (YoutubeVideo, bool, error) {
	args := r.Called(ctx, videoID)

	var v YoutubeVideo
	if args.Get(0) != nil {
		v = args.Get(0).(YoutubeVideo)
	}

	return v, args.Get(1).(bool), args.Error(2)
}

func (r *MockVideoRepository) GetVideos(ctx context.Context, sinceTime time.Time, count int) ([]YoutubeVideo, error) {
	args := r.Called(ctx, sinceTime, count)

	var videos []YoutubeVideo
	if args.Get(0) != nil {
		videos = args.Get(0).([]YoutubeVideo)
	}

	return videos, args.Error(1)
}

func (r *MockVideoRepository) StoreVideoStats(ctx context.Context, stats YoutubeVideoStats) (YoutubeVideoStats, error) {
	args := r.Called(ctx, stats)

	var vs YoutubeVideoStats
	if args.Get(0) != nil {
		vs = args.Get(0).(YoutubeVideoStats)
	}

	return vs, args.Error(1)
}

type internalMockVideoRepository struct {
	MockVideoRepository
}

func (r *internalMockVideoRepository) createVideo(ctx context.Context, q helpers.Querier, video YoutubeVideo) (YoutubeVideo, error) {
	args := r.Called(ctx, q, video)

	var v YoutubeVideo
	if args.Get(0) != nil {
		v = args.Get(0).(YoutubeVideo)
	}

	return v, args.Error(1)
}

func (r *internalMockVideoRepository) getVideo(ctx context.Context, q helpers.Querier, videoID int64) (YoutubeVideo, bool, error) {
	args := r.Called(ctx, q, videoID)

	var v YoutubeVideo
	if args.Get(0) != nil {
		v = args.Get(0).(YoutubeVideo)
	}

	return v, args.Get(1).(bool), args.Error(2)
}

func (r *internalMockVideoRepository) getVideoByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeVideo, bool, error) {
	args := r.Called(ctx, q, externalID)

	var v YoutubeVideo
	if args.Get(0) != nil {
		v = args.Get(0).(YoutubeVideo)
	}

	return v, args.Get(1).(bool), args.Error(2)
}

func (r *internalMockVideoRepository) getVideoStat(ctx context.Context, q helpers.Querier, videoID int64) ([]YoutubeVideoStats, error) {
	args := r.Called(ctx, q, videoID)

	var out []YoutubeVideoStats
	if args.Get(0) != nil {
		out = args.Get(0).([]YoutubeVideoStats)
	}

	return out, args.Error(1)
}

func (r *internalMockVideoRepository) getVideosByChannelPaginated(ctx context.Context, q helpers.Querier, channelID int64, offset int, limit int) ([]YoutubeVideo, error) {
	args := r.Called(ctx, q, channelID, offset, limit)

	var out []YoutubeVideo
	if args.Get(0) != nil {
		out = args.Get(0).([]YoutubeVideo)
	}

	return out, args.Error(1)
}

func (r *internalMockVideoRepository) getVideosByChannelCount(ctx context.Context, q helpers.Querier, channelID int64) (int, error) {
	args := r.Called(ctx, q, channelID)

	return args.Get(0).(int), args.Error(1)
}

func (r *internalMockVideoRepository) updateVideoCheckedAt(ctx context.Context, q helpers.Querier, videoID int64, checkedAt time.Time) error {
	args := r.Called(ctx, q, videoID, checkedAt)
	return args.Error(0)
}

type internalMockChannelRepository struct {
	MockChannelRepository
}

func (r *internalMockChannelRepository) createChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (YoutubeChannel, error) {
	args := r.Called(ctx, q, channel)

	return args.Get(0).(YoutubeChannel), args.Error(1)
}

func (r *MockChannelRepository) getChannelsPaginated(ctx context.Context, q helpers.Querier, offset int, limit int) ([]YoutubeChannel, error) {
	args := r.Called(ctx, q, offset, limit)

	var out []YoutubeChannel
	if args.Get(0) != nil {
		out = args.Get(0).([]YoutubeChannel)
	}

	return out, args.Error(1)
}

func (r *MockChannelRepository) getChannelsCount(ctx context.Context, q helpers.Querier) (int, error) {
	args := r.Called(ctx, q)

	return args.Get(0).(int), args.Error(1)
}

func (r *internalMockChannelRepository) getChannel(ctx context.Context, q helpers.Querier, channelID int64) (YoutubeChannel, bool, error) {
	args := r.Called(ctx, q, channelID)

	var out YoutubeChannel
	if args.Get(0) != nil {
		out = args.Get(0).(YoutubeChannel)
	}

	return out, args.Get(1).(bool), args.Error(2)
}

func (r *internalMockChannelRepository) getChannelStat(ctx context.Context, q helpers.Querier, channelID int64) ([]YoutubeChannelStats, error) {
	args := r.Called(ctx, q, channelID)

	var out []YoutubeChannelStats
	if args.Get(0) != nil {
		out = args.Get(0).([]YoutubeChannelStats)
	}

	return out, args.Error(1)
}

func (r *internalMockChannelRepository) getChannelByExternalId(ctx context.Context, q helpers.Querier, externalID string) (YoutubeChannel, bool, error) {
	args := r.Called(ctx, q, externalID)

	var out YoutubeChannel
	if args.Get(0) != nil {
		out = args.Get(0).(YoutubeChannel)
	}

	return out, args.Get(1).(bool), args.Error(2)
}

type EmptyTx struct {
	pgx.Tx
}

func (e EmptyTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return e, nil
}

func (e EmptyTx) Commit(ctx context.Context) error {
	return nil
}

func (e EmptyTx) Rollback(ctx context.Context) error {
	return nil
}

func (e EmptyTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errUnimplemented
}

func (e EmptyTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (e EmptyTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (e EmptyTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errUnimplemented
}

func (e EmptyTx) Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error) {
	return pgconn.CommandTag{}, errUnimplemented
}

func (e EmptyTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errUnimplemented
}

func (e EmptyTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (e EmptyTx) Conn() *pgx.Conn {
	return nil
}

type MockTx struct {
	EmptyTx
	mock.Mock
}

type MockTxController struct {
	mock.Mock
}

func (m *MockTxController) Begin(ctx context.Context) (pgx.Tx, error) {
	return &MockTx{}, nil
}

type MockVideoService struct {
	mock.Mock
}

func (s *MockVideoService) CreateVideo(ctx context.Context, video YoutubeVideo) (YoutubeVideo, bool, error) {
	args := s.Called(ctx, video)

	var v YoutubeVideo
	if args.Get(0) != nil {
		v = args.Get(0).(YoutubeVideo)
	}

	return v, args.Get(1).(bool), args.Error(2)
}

func (s *MockVideoService) GetVideoStats(ctx context.Context, videoID int64) (YoutubeVideoStatsInfo, bool, error) {
	args := s.Called(ctx, videoID)

	var info YoutubeVideoStatsInfo
	if args.Get(0) != nil {
		info = args.Get(0).(YoutubeVideoStatsInfo)
	}

	return info, args.Get(1).(bool), args.Error(2)
}

func (s *MockVideoService) GetVideosByChannel(ctx context.Context, channelID int64, page int, pageSize int) (YoutubeVideosInfo, bool, error) {
	args := s.Called(ctx, channelID, page, pageSize)

	var info YoutubeVideosInfo
	if args.Get(0) != nil {
		info = args.Get(0).(YoutubeVideosInfo)
	}

	return info, args.Get(1).(bool), args.Error(2)
}
