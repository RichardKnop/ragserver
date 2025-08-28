package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
)

type insertFilesQuery struct {
	files []*ragserver.File
}

func (q insertFilesQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	sql := `
		with cte as (
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?, ?)
	`
	args := make([]any, 0, len(q.files)*13)
	args = append(
		args,
		q.files[0].ID,
		q.files[0].FileName,
		q.files[0].ContentType,
		q.files[0].Extension,
		q.files[0].Size,
		q.files[0].Hash,
		q.files[0].Embedder,
		q.files[0].Retriever,
		q.files[0].Location,
		q.files[0].Status,
		q.files[0].StatusMessage,
		q.files[0].CreatedAt,
		q.files[0].UpdatedAt,
	)
	for i := range q.files[1:] {
		sql += `, (?, ?, ?, ?, ?, ?, ?, ?, ?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?, ?)`
		args = append(
			args,
			q.files[i+1].ID,
			q.files[i+1].FileName,
			q.files[i+1].ContentType,
			q.files[i+1].Extension,
			q.files[i+1].Size,
			q.files[i+1].Hash,
			q.files[i+1].Embedder,
			q.files[i+1].Retriever,
			q.files[i+1].Location,
			q.files[i+1].Status,
			q.files[i+1].StatusMessage,
			q.files[i+1].CreatedAt,
			q.files[i+1].UpdatedAt,
		)
	}
	sql += `
		)
		insert into "file" (
			"id", 
			"file_name", 
			"content_type", 
			"extension",
			"file_size", 
			"file_hash",
			"embedder",
			"retriever",
			"location",
			"status",
			"status_message",
			"created_at",
			"updated_at"
		)
		select * 
		from cte
		where 1
		on conflict("id") do update set
			"file_name"=excluded."file_name",
			"content_type"=excluded."content_type",
			"extension"=excluded."extension",
			"file_size"=excluded."file_size",
			"file_hash"=excluded."file_hash",
			"embedder"=excluded."embedder",
			"retriever"=excluded."retriever",
			"location"=excluded."location",
			"status"=excluded."status",
			"status_message"=excluded."status_message",
			"updated_at"=excluded."updated_at"	
	`

	return sql, args
}

func (a *Adapter) SaveFiles(ctx context.Context, files ...*ragserver.File) error {
	if len(files) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := insertFilesQuery{files: files}.SQL()

		stmt, err := tx.Prepare(query)
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
	}); err != nil {
		return err
	}

	return nil
}

type selectFilesQuery struct{}

func (q selectFilesQuery) SQL() (string, []any) {
	return `
		select 
			f."id",
			f."file_name", 
			f."content_type", 
			f."extension", 
			f."file_size", 
			f."file_hash",
			f."embedder",
			f."retriever",
			f."location",
			fs."name" AS "status",
			f."status_message",
			f."created_at",
			f."updated_at"
		from "file" f
		inner join "file_status" fs on f."status" = fs."id"
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
			var aFile = new(ragserver.File)
			aFile, err = scanFile(rows)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
			files = append(files, aFile)
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
		clauses = append(clauses, `f."embedder" = ?`)
		args = append(args, filter.Embedder)
	}

	if filter.Retriever != "" {
		clauses = append(clauses, `f."retriever" = ?`)
		args = append(args, filter.Retriever)
	}

	if filter.Status != "" {
		clauses = append(clauses, `fs."name" = ?`)
		args = append(args, filter.Status)
	}

	if !filter.LastUpdatedBefore.T.IsZero() {
		clauses = append(clauses, `f."updated_at" < ?`)
		args = append(args, filter.LastUpdatedBefore)
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
			f."id",
			f."file_name", 
			f."content_type", 
			f."extension", 
			f."file_size", 
			f."file_hash",
			f."embedder",
			f."retriever",
			f."location",
			fs."name" AS "status",
			f."status_message",
			f."created_at",
			f."updated_at"
		FROM "file" f
		INNER JOIN "file_status" fs ON f."status" = fs."id"		
		WHERE f."id" = ?
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
			return fmt.Errorf("scan failed: %w", err)
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
	var aFile = new(ragserver.File)

	if err := row.Scan(
		&aFile.ID,
		&aFile.FileName,
		&aFile.ContentType,
		&aFile.Extension,
		&aFile.Size,
		&aFile.Hash,
		&aFile.Embedder,
		&aFile.Retriever,
		&aFile.Location,
		&aFile.Status,
		&aFile.StatusMessage,
		&aFile.CreatedAt,
		&aFile.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ragserver.ErrNotFound
		}
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return aFile, nil
}
