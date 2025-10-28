package helpers

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Querier interface that both pgx.Conn and pgx.Tx satisfy
type Querier interface {
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}
