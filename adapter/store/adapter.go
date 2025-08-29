package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Adapter struct {
	db *sql.DB
}

type Option func(*Adapter)

func New(db *sql.DB, options ...Option) *Adapter {
	a := &Adapter{
		db: db,
	}

	for _, o := range options {
		o(a)
	}

	return a
}

type Scannable interface {
	Scan(dest ...any) error
}

type Query interface {
	SQL() (string, []any)
}

func execBatchInsertQuery(ctx context.Context, tx *sql.Tx, q Query) error {
	sql, args := q.SQL()
	stmt, err := tx.Prepare(sql)
	if err != nil {
		return fmt.Errorf("prepare statement failed: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("exec context failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}

func execQuery(ctx context.Context, tx *sql.Tx, q Query) error {
	sql, args := q.SQL()
	return exec(ctx, tx, sql, args...)
}

func exec(ctx context.Context, tx *sql.Tx, sql string, args ...any) error {
	stmt, err := tx.Prepare(sql)
	if err != nil {
		return fmt.Errorf("prepare statement failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("exec context failed: %w", err)
	}

	return nil
}
