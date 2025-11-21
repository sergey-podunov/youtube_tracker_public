//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"testing"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ChannelRepoTestSuite struct {
	BaseChannelRepoTestSuite
}

func TestChannelRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelRepoTestSuite))
}

func (suite *ChannelRepoTestSuite) insertChannel(channel YoutubeChannel) int64 {
	t := suite.T()
	ctx := suite.ctx

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
	) RETURNING youtube_channel_id`

	var insertedID int64
	err := suite.tx.QueryRow(ctx, query, channel.Name, channel.ExternalID, channel.CreatedAt).Scan(&insertedID)
	require.NoError(t, err)

	return insertedID
}

func (suite *ChannelRepoTestSuite) TestCreateChannel() {
	t := suite.T()
	ctx := suite.ctx

	channel, err := suite.repository.createChannel(
		ctx,
		suite.tx,
		YoutubeChannel{
			ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			Name:       "Google Developers",
		})
	require.NoError(t, err)
	require.NotNil(t, channel.YoutubeChannelId)

	var actualChannel YoutubeChannel
	err =
		suite.tx.QueryRow(ctx, "SELECT * FROM youtube_channel WHERE youtube_channel_id = $1", channel.YoutubeChannelId).
			Scan(&actualChannel.YoutubeChannelId, &actualChannel.ExternalID, &actualChannel.Name, &actualChannel.CreatedAt, &actualChannel.CheckedAt)
	require.NoError(t, err)

	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
	assert.Equal(t, "Google Developers", actualChannel.Name)
	assert.False(t, actualChannel.CreatedAt.IsZero())
	assert.Equal(t, channel.CreatedAt, actualChannel.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannel() {
	t := suite.T()
	ctx := suite.ctx

	createdAt := helpers.ParseTime("2005-08-15T15:52:01Z")
	channelToInsert := YoutubeChannel{ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw", Name: "Google Developers", CreatedAt: createdAt}
	insertedID := suite.insertChannel(channelToInsert)

	actualChannel, ok, err := suite.repository.GetChannel(ctx, insertedID)
	require.NoError(t, err)

	assert.True(t, ok)
	assert.Equal(t, insertedID, actualChannel.YoutubeChannelId)
	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
	assert.Equal(t, "Google Developers", actualChannel.Name)
	assert.Equal(t, createdAt, actualChannel.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannelNotFound() {
	t := suite.T()
	ctx := suite.ctx

	actualChannel, ok, err := suite.repository.GetChannel(ctx, 1234789)
	assert.Nil(t, err)
	assert.Equal(t, YoutubeChannel{}, actualChannel)
	assert.False(t, ok)
}

func (suite *ChannelRepoTestSuite) TestGetChannelByExternalId() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	createdAt := helpers.ParseTime("2005-08-15T15:52:01Z")
	channelToInsert := YoutubeChannel{ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw", Name: "Google Developers", CreatedAt: createdAt}
	insertedID := suite.insertChannel(channelToInsert)

	actualChannel, ok, err := suite.repository.getChannelByExternalId(ctx, tx, "UC-lHJZR3Gqxm24_Vd_AJ5Yw")
	require.NoError(t, err)

	assert.True(t, ok)
	assert.Equal(t, insertedID, actualChannel.YoutubeChannelId)
	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
	assert.Equal(t, "Google Developers", actualChannel.Name)
	assert.Equal(t, createdAt, actualChannel.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannelByExternalIdNotFound() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	actualChannel, ok, err := suite.repository.getChannelByExternalId(ctx, tx, "UC-lHJZR3Gqxm24_Vd_AJ5Yw")

	assert.NoError(t, err)
	assert.Equal(t, YoutubeChannel{}, actualChannel)
	assert.False(t, ok)
}

func (suite *ChannelRepoTestSuite) TestStoreSubscriptionsCount() {
	t := suite.T()
	ctx := suite.ctx

	channel, err := suite.repository.createChannel(
		ctx,
		suite.tx,
		YoutubeChannel{
			ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			Name:       "Google Developers",
		})
	require.NoError(t, err)
	require.NotNil(t, channel.YoutubeChannelId)

	channelStats, err := suite.repository.StoreSubscriptionsCount(
		ctx,
		YoutubeChannelStats{
			YoutubeChannelID: channel.YoutubeChannelId,
			SubscribersCount: 3,
		})
	require.NoError(t, err)
	require.NotNil(t, channel.YoutubeChannelId)

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
	err = suite.tx.QueryRow(ctx, query, channel.YoutubeChannelId).
		Scan(
			&actualChannelStats.YoutubeChannelID,
			&actualChannelStats.YoutubeChannelStatID,
			&actualChannelStats.SubscribersCount,
			&actualChannelStats.CreatedAt,
		)
	require.NoError(t, err)

	assert.Equal(t, channelStats.YoutubeChannelID, actualChannelStats.YoutubeChannelID)
	assert.Equal(t, channelStats.YoutubeChannelStatID, actualChannelStats.YoutubeChannelStatID)
	assert.Equal(t, channelStats.SubscribersCount, actualChannelStats.SubscribersCount)
	assert.Equal(t, int64(3), actualChannelStats.SubscribersCount)
	assert.False(t, actualChannelStats.CreatedAt.IsZero())
	assert.Equal(t, channelStats.CreatedAt, actualChannelStats.CreatedAt)
}

func (suite *ChannelRepoTestSuite) TestGetChannelsPaginated() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx
	
	createdAt := helpers.ParseTime("2005-08-15T15:52:01Z")
	
	channelToInsert := YoutubeChannel{ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw", Name: "Google Developers", CreatedAt: createdAt}
	_ = suite.insertChannel(channelToInsert)
	
	channelToInsert = YoutubeChannel{ExternalID: "test_channel_youtube_id", Name: "Test channel", CreatedAt: createdAt}
	insertedID := suite.insertChannel(channelToInsert)

	actualChannels, err := suite.repository.getChannelsPaginated(ctx, tx, 1, 1)
	require.NoError(t, err)

	assert.Equal(t, 1, len(actualChannels))
	actualChannel := actualChannels[0]
	assert.Equal(t, insertedID, actualChannel.YoutubeChannelId)
}

func (suite *ChannelRepoTestSuite) TestGetChannelsPaginated_emptyDB() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	actualChannels, err := suite.repository.getChannelsPaginated(ctx, tx, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, 0, len(actualChannels))
}

func (suite *ChannelRepoTestSuite) TestGetChannelsCount() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	createdAt := helpers.ParseTime("2005-08-15T15:52:01Z")
	channelToInsert := YoutubeChannel{ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw", Name: "Google Developers", CreatedAt: createdAt}
	_ = suite.insertChannel(channelToInsert)
	
	actualChannelsCount, err := suite.repository.getChannelsCount(ctx, tx)
	require.NoError(t, err)

	assert.Equal(t, 1, actualChannelsCount)
}
