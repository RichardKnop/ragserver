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
func (a *Adapter) Transactional(ctx context.Context, options *sql.TxOptions, fn func(ctx context.Context) error) (finalErr error) {
	_, ok := ctx.Value(transactionKey()).(*sql.Tx)
	if ok {
		return fn(ctx)
	}

	tx, err := a.db.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			finalErr = errors.Join(fmt.Errorf("rollback: %w", err), finalErr)
		}
	}()

	ctx = context.WithValue(ctx, transactionKey(), tx)
	if err := fn(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

type Transactional func(ctx context.Context, tx *sql.Tx) error

func (a *Adapter) inTxDo(ctx context.Context, options *sql.TxOptions, fn Transactional) error {
	return a.inChainedTxDo(ctx, options, fn)
}

func (a *Adapter) inChainedTxDo(ctx context.Context, options *sql.TxOptions, fn Transactional) error {
	parentTx, ok := ctx.Value(transactionKey()).(*sql.Tx)
	if ok {
		return fn(ctx, parentTx)
	}

	tx, err := a.db.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		// rollback if we encounter an error
		// if transaction is already committed, this will return sql.ErrTxDone
		_ = tx.Rollback()
	}()

	// add transaction to context so child functions can retrieve it
	ctx = context.WithValue(ctx, transactionKey(), tx)

	if err := fn(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
