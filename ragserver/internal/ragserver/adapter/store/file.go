package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type insertFileQuery struct {
	*ragserver.File
}

func (q insertFileQuery) SQL() (string, []any) {
	return `
		INSERT INTO "file" (
			"id", 
			"file_name", 
			"mime_type", 
			"extension",
			"file_size", 
			"created_at"
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`, []any{q.ID, q.FileName, q.MimeType, q.Extension, q.Size, q.CreatedAt}
}

func (a *Adapter) SaveFile(ctx context.Context, file *ragserver.File) error {
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := insertFileQuery{file}.SQL()

		stmt, err := tx.Prepare(query)
		if err != nil {
			return fmt.Errorf("prepare statement failed: %w", err)
		}
		defer stmt.Close()

		result, err := stmt.ExecContext(ctx, args...)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected failed: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("no rows affected")
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type selectFilesQuery struct {
}

func (q selectFilesQuery) SQL() (string, []any) {
	return `
		SELECT 
			"id", 
			"file_name", 
			"mime_type", 
			"extension", 
			"file_size", 
			"created_at"
		FROM "file"
		ORDER BY "created_at" DESC
	`, nil
}

func (a *Adapter) ListFiles(ctx context.Context) ([]*ragserver.File, error) {
	var files []*ragserver.File

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := selectFilesQuery{}.SQL()

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var file ragserver.File
			if err := rows.Scan(
				&file.ID,
				&file.FileName,
				&file.MimeType,
				&file.Extension,
				&file.Size,
				&file.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
			files = append(files, &file)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

type findFileQuery struct {
	id ragserver.FileID
}

func (q findFileQuery) SQL() (string, []any) {
	return `
		SELECT 
			"id", 
			"file_name", 
			"mime_type", 
			"extension", 
			"file_size", 
			"created_at"
		FROM "file" where "id" = ?
	`, []any{q.id}
}

func (a *Adapter) FindFile(ctx context.Context, id ragserver.FileID) (*ragserver.File, error) {
	var file = new(ragserver.File)
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := findFileQuery{id: id}.SQL()

		stmt, err := tx.Prepare(query)
		if err != nil {
			return fmt.Errorf("prepare statement failed: %w", err)
		}
		defer stmt.Close()

		row := stmt.QueryRowContext(ctx, args...)
		if err := row.Scan(
			&file.ID,
			&file.FileName,
			&file.MimeType,
			&file.Extension,
			&file.Size,
			&file.CreatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ragserver.ErrNotFound
			}
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return file, nil
}
