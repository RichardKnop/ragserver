package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type contextKey string

func transactionKey() contextKey {
	return contextKey("tx")
}

// Transactional is a helper function that executes a function within a database transaction.
func (a *Adapter) Transactional(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) (finalErr error) {
	_, ok := ctx.Value(transactionKey()).(*sql.Tx)
	// TODO - check for options being compatible with existing transaction (isolation level, etc.)
	if ok {
		return fn(ctx)
	}

	tx, err := a.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			finalErr = errors.Join(fmt.Errorf("rollback: %w", err), finalErr)
		}
	}()

	if err := fn(context.WithValue(ctx, transactionKey(), tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (a *Adapter) inTxDo(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	tx, ok := ctx.Value(transactionKey()).(*sql.Tx)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}

	return fn(ctx, tx)
}
