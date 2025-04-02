//go:build database

package stats

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ChannelRepoTestSuite struct {
	suite.Suite
	pgContainer *helpers.PostgresContainer
	repository  *YoutubeChannelRepository
	ctx         context.Context
}

func (suite *ChannelRepoTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	pgContainer, err := helpers.CreatePostgresContainer(suite.ctx)
	if err != nil {
		log.Fatal(err)
	}
	suite.pgContainer = pgContainer

	repository, err := NewChannelRepository(suite.ctx, suite.pgContainer.ConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	suite.repository = repository
}

func (suite *ChannelRepoTestSuite) TearDownSuite() {
	if err := suite.pgContainer.Terminate(suite.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (suite *ChannelRepoTestSuite) TestCreateChannel() {
	t := suite.T()
	ctx := suite.ctx

	channel, err := suite.repository.CreateChannel(
		ctx,
		YoutubeChannel{
			ExternalId: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			Name:       "Google Developers",
		})
	require.NoError(t, err)
	require.NotNil(t, channel.YoutubeChannelId)

	conn, _ := pgx.Connect(ctx, suite.pgContainer.ConnectionString)

	defer conn.Close(ctx)
	var actualChannel YoutubeChannel
	err =
		conn.QueryRow(ctx, "SELECT * FROM youtube_channel WHERE youtube_channel_id = $1", channel.YoutubeChannelId).
			Scan(&actualChannel.YoutubeChannelId, &actualChannel.ExternalId, &actualChannel.Name, &actualChannel.CreatedAt)

	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalId)
	assert.Equal(t, "Google Developers", actualChannel.Name)
	assert.False(t, actualChannel.CreatedAt.IsZero())
	assert.Equal(t, channel.CreatedAt, actualChannel.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannel() {
	t := suite.T()
	ctx := suite.ctx

	conn, _ := pgx.Connect(ctx, suite.pgContainer.ConnectionString)
	defer conn.Close(ctx)

	query := `
	INSERT INTO youtube_channel (
		channel_name, 
		external_id, 
		created_at
	) 
	VALUES (
		$1, 
		$2, 
		$3
	) 
	RETURNING youtube_channel_id
`
	createdAt, _ := time.Parse(time.RFC3339, "2005-08-15T15:52:01Z")
	var insertedID int64
	err := conn.QueryRow(ctx, query, "Google Developers", "UC-lHJZR3Gqxm24_Vd_AJ5Yw", createdAt).Scan(&insertedID)
	require.NoError(t, err)

	actualChannel, err := suite.repository.GetChannel(ctx, insertedID)
	require.NoError(t, err)

	assert.Equal(t, insertedID, actualChannel.YoutubeChannelId)
	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalId)
	assert.Equal(t, "Google Developers", actualChannel.Name)
	assert.Equal(t, createdAt, actualChannel.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannelNotFound() {
	t := suite.T()
	ctx := suite.ctx

	actualChannel, err := suite.repository.GetChannel(ctx, 1234789)
	assert.Error(t, err)
	assert.Nil(t, actualChannel)
}

func (suite *ChannelRepoTestSuite) TestStoreSubscriptionsCount() {
	t := suite.T()
	ctx := suite.ctx

	channel, err := suite.repository.CreateChannel(
		ctx,
		YoutubeChannel{
			ExternalId: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			Name:       "Google Developers",
		})

	channelStats, err := suite.repository.StoreSubscriptionsCount(
		ctx,
		YoutubeChannelStats{
			YoutubeChannelId: channel.YoutubeChannelId,
			SubscribersCount: 3,
		})
	require.NoError(t, err)
	require.NotNil(t, channel.YoutubeChannelId)

	conn, _ := pgx.Connect(ctx, suite.pgContainer.ConnectionString)
	defer conn.Close(ctx)

	query := `
	SELECT 
		youtube_channel_id, 
		youtube_channel_stat_id, 
		subscribers_count, 
		created_at 
	FROM 
		youtube_channel_stat 
	WHERE 
		youtube_channel_id = $1
`
	var actualChannelStats YoutubeChannelStats
	err = conn.QueryRow(ctx, query, channel.YoutubeChannelId).
		Scan(
			&actualChannelStats.YoutubeChannelId,
			&actualChannelStats.YoutubeChannelStatId,
			&actualChannelStats.SubscribersCount,
			&actualChannelStats.CreatedAt,
		)
	require.NoError(t, err)

	assert.Equal(t, channelStats.YoutubeChannelId, actualChannelStats.YoutubeChannelId)
	assert.Equal(t, channelStats.YoutubeChannelStatId, actualChannelStats.YoutubeChannelStatId)
	assert.Equal(t, channelStats.SubscribersCount, actualChannelStats.SubscribersCount)
	assert.Equal(t, int64(3), actualChannelStats.SubscribersCount)
	assert.False(t, actualChannelStats.CreatedAt.IsZero())
	assert.Equal(t, channelStats.CreatedAt, actualChannelStats.CreatedAt)
}

func TestChannelRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelRepoTestSuite))
}
