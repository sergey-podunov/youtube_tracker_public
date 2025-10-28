//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"context"
	"log"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/suite"
)

type BaseChannelRepoTestSuite struct {
	suite.Suite
	pgContainer *helpers.PostgresContainer
	repository  *YoutubeChannelRepository
	ctx         context.Context
	conn        *pgx.Conn // The main connection for the suite
	tx          pgx.Tx    // The transaction for the current test
}

func (suite *BaseChannelRepoTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	t := suite.T()

	pgContainer, err := helpers.CreatePostgresContainer(suite.ctx)
	if err != nil {
		t.Fatal(err)
	}

	suite.pgContainer = pgContainer

	conn, err := connectWithTrace(suite.ctx, suite.pgContainer.ConnectionString)
	if err != nil {
		t.Fatal(err)
	}

	suite.conn = conn
}

func (suite *BaseChannelRepoTestSuite) TearDownSuite() {
	if suite.conn != nil {
		_ = suite.conn.Close(suite.ctx)
	}

	if err := suite.pgContainer.Terminate(suite.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (suite *BaseChannelRepoTestSuite) SetupTest() {
	t := suite.T()

	tx, err := suite.conn.Begin(suite.ctx)
	if err != nil {
		t.Fatal(err)
	}

	suite.tx = tx

	// Reset repository with transaction for each test
	suite.repository = &YoutubeChannelRepository{querier: tx}
}

func (suite *BaseChannelRepoTestSuite) TearDownTest() {
	t := suite.T()

	if suite.tx != nil {
		if err := suite.tx.Rollback(suite.ctx); err != nil {
			t.Logf("Error rolling back transaction: %v", err)
		}
	}
}

//nolint:unused
func insertChannels(ctx context.Context, tx pgx.Tx, expectedChannels []YoutubeChannel) error {
	query := `
			INSERT INTO youtube_channel (
					channel_name,
					external_id,
					created_at,
			        checked_at                    
			) VALUES (
				$1,
				$2,
				$3,
			    $4
			)`

	for _, channel := range expectedChannels {
		_, err := tx.Exec(ctx, query, channel.Name, channel.ExternalId, channel.CreatedAt, channel.CheckedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

type stdLogger struct{}

func (l stdLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	log.Printf("[pgx %s] %s - %v", level, msg, data)
}

func connectWithTrace(ctx context.Context, connStr string) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	cfg.Tracer = &tracelog.TraceLog{
		Logger:   stdLogger{},
		LogLevel: tracelog.LogLevelTrace,
	}

	return pgx.ConnectConfig(ctx, cfg)
}
