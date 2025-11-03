package stats

import (
	"context"
	"errors"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

var errUnimplemented = errors.New("unimplemented")

type MockChannelRepository struct {
	mock.Mock
}

func (r *MockChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (*YoutubeChannel, bool, error) {
	args := r.Called(ctx, youtubeChannelId)

	var ch *YoutubeChannel
	if args.Get(0) != nil {
		ch = args.Get(0).(*YoutubeChannel)
	}

	return ch, args.Get(1).(bool), args.Error(2)
}

func (r *MockChannelRepository) StoreSubscriptionsCount(ctx context.Context, channelStats YoutubeChannelStats) (*YoutubeChannelStats, error) {
	args := r.Called(ctx, channelStats)

	var cs *YoutubeChannelStats
	if args.Get(0) != nil {
		cs = args.Get(0).(*YoutubeChannelStats)
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

type internalMockChannelRepository struct {
	MockChannelRepository
}

func (r *internalMockChannelRepository) createChannel(ctx context.Context, q helpers.Querier, channel YoutubeChannel) (YoutubeChannel, error) {
	args := r.Called(ctx, q, channel)

	return args.Get(0).(YoutubeChannel), args.Error(1)
}

func (r *internalMockChannelRepository) getChannel(ctx context.Context, q helpers.Querier, channelID int64) (*YoutubeChannel, bool, error) {
	args := r.Called(ctx, q, channelID)

	var out *YoutubeChannel
	if args.Get(0) != nil {
		out = args.Get(0).(*YoutubeChannel)
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
