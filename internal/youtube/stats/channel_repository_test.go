//go:build integration

package stats

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"testing"
	"time"
	"youtube_tracker/internal/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ChannelRepoTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	repository  *ChannelRepository
	ctx         context.Context
}

func (suite *ChannelRepoTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	pgContainer, err := testhelpers.CreatePostgresContainer(suite.ctx)
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

	createdAt, err := time.Parse(time.RFC3339, "2005-08-15T15:52:01Z")
	channel, err := suite.repository.createChannel(suite.ctx,
		YoutubeChannel{
			externalId: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			name:       "Google Developers",
			createdAt:  createdAt,
		})
	assert.NoError(t, err)
	assert.NotNil(t, channel.youtubeChannelId)

	conn, _ := pgx.Connect(suite.ctx, suite.pgContainer.ConnectionString)

	defer conn.Close(suite.ctx)
	var actualChannel YoutubeChannel
	err =
		conn.QueryRow(suite.ctx, "SELECT * FROM youtube_channel WHERE youtube_channel_id = $1", channel.youtubeChannelId).
			Scan(&actualChannel.youtubeChannelId, &actualChannel.externalId, &actualChannel.name, &actualChannel.createdAt)

	assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.externalId)
	assert.Equal(t, "Google Developers", actualChannel.name)
	assert.Equal(t, createdAt, actualChannel.createdAt)
}

func TestChannelRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelRepoTestSuite))
}
