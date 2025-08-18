package store

import (
	"context"
	"database/sql"
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
	if err := a.inTxDo(ctx, func(ctx context.Context, tx *sql.Tx) error {
		sql, args := insertFileQuery{file}.SQL()

		stmt, err := tx.Prepare(sql)
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
	`, nil
}

func (a *Adapter) ListFiles(ctx context.Context) ([]*ragserver.File, error) {
	var files []*ragserver.File

	if err := a.inTxDo(ctx, func(ctx context.Context, tx *sql.Tx) error {
		sql, args := selectFilesQuery{}.SQL()

		rows, err := tx.QueryContext(ctx, sql, args...)
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
