package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

type insertFilesQuery struct {
	files []*ragserver.File
}

func (q insertFilesQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	sql := `
		INSERT INTO "file" (
			"id", 
			"file_name", 
			"mime_type", 
			"extension",
			"file_size", 
			"file_hash",
			"embedder",
			"retriever",
			"created_at"
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	args := make([]any, 0, len(q.files)*8)
	args = append(
		args,
		q.files[0].ID,
		q.files[0].FileName,
		q.files[0].MimeType,
		q.files[0].Extension,
		q.files[0].Size,
		q.files[0].Hash,
		q.files[0].Embedder,
		q.files[0].Retriever,
		q.files[0].CreatedAt,
	)
	for i := range q.files[1:] {
		sql += ", (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(
			args,
			q.files[i+1].ID,
			q.files[i+1].FileName,
			q.files[i+1].MimeType,
			q.files[i+1].Extension,
			q.files[i+1].Size,
			q.files[i+1].Hash,
			q.files[i+1].Embedder,
			q.files[i+1].Retriever,
			q.files[i+1].CreatedAt,
		)
	}

	return sql, args
}

func (a *Adapter) SaveFiles(ctx context.Context, files ...*ragserver.File) error {
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := insertFilesQuery{files: files}.SQL()

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

type selectFilesQuery struct{}

func (q selectFilesQuery) SQL() (string, []any) {
	return `
		SELECT 
			"id", 
			"file_name", 
			"mime_type", 
			"extension", 
			"file_size", 
			"file_hash",
			"embedder",
			"retriever",
			"created_at"
		FROM "file"
	`, nil
}

func (a *Adapter) ListFiles(ctx context.Context, filter ragserver.FileFilter, partial authz.Partial) ([]*ragserver.File, error) {
	var files []*ragserver.File

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := selectFilesQuery{}.SQL()

		// Add where clauses from the filter and/or partial if any
		where, whereArgs := fileFilterClauses(filter)
		partialClauses, partialArgs := partial.SQL()
		if partialClauses != "" {
			if where == "" {
				where += partialClauses
			} else {
				where += " AND " + partialClauses
			}

			whereArgs = append(whereArgs, partialArgs...)
		}
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}

		// Add order by clause
		query += ` ORDER BY "created_at" DESC`

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var file *ragserver.File
			file, err = scanFile(rows)
			if err != nil {
				return err
			}
			files = append(files, file)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func fileFilterClauses(filter ragserver.FileFilter) (string, []any) {
	var (
		clauses = []string{}
		args    = []any{}
	)

	if filter.Embedder != "" {
		clauses = append(clauses, "embedder = ?")
		args = append(args, filter.Embedder)
	}

	if filter.Retriever != "" {
		clauses = append(clauses, "retriever = ?")
		args = append(args, filter.Retriever)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
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
			"file_hash",
			"embedder",
			"retriever",
			"created_at"
		FROM "file" where "id" = ?
	`, []any{q.id}
}

func (a *Adapter) FindFile(ctx context.Context, id ragserver.FileID, partial authz.Partial) (*ragserver.File, error) {
	var file *ragserver.File
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := findFileQuery{id: id}.SQL()

		// Add where clauses from the partial if any
		partialClauses, partialArgs := partial.SQL()
		if partialClauses != "" {
			query += " AND " + partialClauses

			args = append(args, partialArgs...)
		}

		stmt, err := tx.Prepare(query)
		if err != nil {
			return fmt.Errorf("prepare statement failed: %w", err)
		}
		defer stmt.Close()

		row := stmt.QueryRowContext(ctx, args...)
		file, err = scanFile(row)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return file, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFile(row scannable) (*ragserver.File, error) {
	file := new(ragserver.File)
	if err := row.Scan(
		&file.ID,
		&file.FileName,
		&file.MimeType,
		&file.Extension,
		&file.Size,
		&file.Hash,
		&file.Embedder,
		&file.Retriever,
		&file.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ragserver.ErrNotFound
		}
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	return file, nil
}
