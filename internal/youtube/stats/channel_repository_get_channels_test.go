//go:build database

package stats

import (
	"log"
	"testing"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ChannelRepoGetChannelsTestSuite struct {
	BaseChannelRepoTestSuite
}

func (suite *ChannelRepoGetChannelsTestSuite) TestGetChannels_testReturnObject() {
	t := suite.T()
	ctx := suite.ctx
	tx := suite.tx

	createdAt := helpers.ParseTime("2025-10-12T05:06:07Z")
	expectedChannels := []YoutubeChannel{
		{ExternalID: "ext_id_1", Name: "Channel 1", CreatedAt: createdAt, CheckedAt: helpers.Ptr(helpers.ParseTime("2025-10-25T05:06:07Z"))},
		{ExternalID: "ext_id_2", Name: "Channel 2", CreatedAt: createdAt},
	}

	if err := insertChannels(ctx, tx, expectedChannels); err != nil {
		t.Fatalf("Error inserting channels: %v", err)
	}

	actualChannels, err := suite.repository.GetChannels(ctx, helpers.ParseTime("2225-10-15T00:00:00Z"), 10)
	require.NoError(t, err)

	assert.ElementsMatch(t, toYoutubeChannelView(expectedChannels), toYoutubeChannelView(actualChannels), "Returned channels do not match")
}

type channelView struct {
	ExternalId string
	Name       string
	CreatedAt  time.Time
	CheckedAt  *time.Time
}

func toYoutubeChannelView(in []YoutubeChannel) []channelView {
	out := make([]channelView, len(in))
	for i, c := range in {
		out[i] = channelView{
			ExternalId: c.ExternalID,
			Name:       c.Name,
			CreatedAt:  c.CreatedAt,
			CheckedAt:  c.CheckedAt,
		}
	}
	return out
}

func (suite *ChannelRepoGetChannelsTestSuite) TestGetChannels() {
	ctx := suite.ctx

	type testCase struct {
		name               string
		channelsBeforeTest []YoutubeChannel
		count              int
		checkedBefore      time.Time
		expectedIds        []string
	}

	createdAt := helpers.ParseTime("2000-01-01T00:00:00Z")
	tests := []testCase{
		{
			name:               "No channels in database",
			channelsBeforeTest: []YoutubeChannel{},
			count:              10,
			checkedBefore:      helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:        []string{},
		},
		{
			name:               "checkedAt is null",
			channelsBeforeTest: []YoutubeChannel{{ExternalID: "id1", Name: "Channel 1", CreatedAt: createdAt}},
			count:              10,
			checkedBefore:      helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:        []string{"id1"},
		},
		{
			name: "matched by checkedAt",
			channelsBeforeTest: []YoutubeChannel{
				{ExternalID: "id2", Name: "Channel 2", CreatedAt: createdAt, CheckedAt: helpers.Ptr(helpers.ParseTime("2025-10-01T00:00:00Z"))},
				{ExternalID: "id3", Name: "Channel 3", CreatedAt: createdAt, CheckedAt: helpers.Ptr(helpers.ParseTime("2025-09-01T00:00:00Z"))},
			},
			count:         10,
			checkedBefore: helpers.ParseTime("2025-09-24T00:00:00Z"),
			expectedIds:   []string{"id3"},
		},
		{
			name: "limit by count",
			channelsBeforeTest: []YoutubeChannel{
				{ExternalID: "id4", Name: "Channel 4", CreatedAt: createdAt},
				{ExternalID: "id5", Name: "Channel 5", CreatedAt: createdAt},
				{ExternalID: "id6", Name: "Channel 6", CreatedAt: createdAt},
			},
			count:         2,
			checkedBefore: helpers.ParseTime("2025-10-24T00:00:00Z"),
			expectedIds:   []string{"id4", "id5"},
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			t := suite.T()

			tx, err := suite.conn.Begin(ctx)
			defer func() {
				if err := tx.Rollback(ctx); err != nil {
					t.Logf("Error rolling back transaction for subtest %s: %v", tc.name, err)
				}
			}()
			require.NoError(t, err)

			subtestRepo := &YoutubeChannelRepository{db: tx}

			if err := insertChannels(ctx, tx, tc.channelsBeforeTest); err != nil {
				t.Fatalf("Error inserting channels for subtest %s: %v", tc.name, err)
			}

			actualChannels, err := subtestRepo.GetChannels(ctx, tc.checkedBefore, tc.count)
			require.NoError(t, err)

			var actualExternalIds []string
			for _, ch := range actualChannels {
				log.Printf("Channel: %v", ch)
				actualExternalIds = append(actualExternalIds, ch.ExternalID)
			}
			assert.ElementsMatch(t, tc.expectedIds, actualExternalIds, "Returned channels do not match expected IDs (ignoring order)")
		})
	}
}

func TestChannelRepoGetChannelsSuite(t *testing.T) {
	suite.Run(t, new(ChannelRepoGetChannelsTestSuite))
}
