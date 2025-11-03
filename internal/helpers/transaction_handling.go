package helpers

import (
	"context"
	"fmt"
	
	"github.com/jackc/pgx/v5"
)

type TxController interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// TxFunc is a function that performs database operations within a pgx transaction.
// It receives a pgx.Tx (the transaction) and should return an error if anything goes wrong.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// RunInTx executes the provided TxFunc within a new pgx transaction from the pool.
// It handles beginning the transaction, committing it on success, and rolling it back
// on error or panic.
func RunInTx(ctx context.Context, txController TxController, fn TxFunc) (err error) {
	tx, err := txController.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(ctx, tx)
	return err
}
