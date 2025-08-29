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

func (a *Adapter) SaveFiles(ctx context.Context, files ...*ragserver.File) error {
	if len(files) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execBatchInsertQuery(ctx, tx, insertFilesQuery{files: files}); err != nil {
			return fmt.Errorf("exec batch insert query failed: %w", err)
		}

		if err := execBatchInsertQuery(ctx, tx, insertFileStatusEventsQuery{files: files}); err != nil {
			return fmt.Errorf("exec batch insert query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type insertFilesQuery struct {
	files []*ragserver.File
}

func (q insertFilesQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	query := `
		with cte as (
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?)
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
		q.files[0].CreatedAt,
		q.files[0].UpdatedAt,
	)
	for i := range q.files[1:] {
		query += `, (?, ?, ?, ?, ?, ?, ?, ?, ?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?)`
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
			q.files[i+1].CreatedAt,
			q.files[i+1].UpdatedAt,
		)
	}
	query += `
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
			"updated_at"=excluded."updated_at"	
	`

	return query, args
}

type insertFileStatusEventsQuery struct {
	files []*ragserver.File
}

func (q insertFileStatusEventsQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	query := `
		with cte as (
			values (?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?)
	`
	args := make([]any, 0, len(q.files)*13)
	args = append(
		args,
		q.files[0].ID,
		q.files[0].Status,
		sql.NullString{String: q.files[0].StatusMessage, Valid: q.files[0].StatusMessage != ""},
		q.files[0].CreatedAt,
	)
	for i := range q.files[1:] {
		query += `, (?, (select "id" from "file_status" fs where fs."name" = ?), ?, ?)`
		args = append(
			args,
			q.files[i+1].ID,
			q.files[i+1].Status,
			sql.NullString{String: q.files[i+1].StatusMessage, Valid: q.files[i+1].StatusMessage != ""},
			q.files[i+1].CreatedAt,
		)
	}
	query += `
		)
		insert into "file_status_evt" (
			"file", 
			"status",
			"message",
			"created_at"
		)
		select * 
		from cte
		where 1
	`

	return query, args
}

func (a *Adapter) ListFiles(ctx context.Context, filter ragserver.FileFilter, partial authz.Partial) ([]*ragserver.File, error) {
	var files []*ragserver.File

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		sql, args := selectFilesQuery{
			filter:  filter,
			partial: partial,
		}.SQL()

		// Add order by clause
		sql += ` order by f."created_at" desc`

		rows, err := tx.QueryContext(ctx, sql, args...)
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

type selectFilesQuery struct {
	filter  ragserver.FileFilter
	partial authz.Partial
}

func (q selectFilesQuery) SQL() (string, []any) {
	query := `
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
			fs."name" as "status",
			fse."message" as "status_message",
			f."created_at",
			f."updated_at"
		from "file" f
		inner join "file_status" fs on f."status" = fs."id"
		inner join "file_status_evt" fse on fse."file" = f."id" and fse."status" = fs."id"
	`
	args := []any{}

	// Add where clauses from the filter and/or partial if any
	where, whereArgs := fileFilterClauses(q.filter)
	partialClauses, partialArgs := q.partial.SQL()
	if partialClauses != "" {
		if where == "" {
			where += partialClauses
		} else {
			where += " and " + partialClauses
		}

		whereArgs = append(whereArgs, partialArgs...)
	}
	if where != "" {
		query += " where " + where
		args = append(args, whereArgs...)
	}

	return query, args
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

func (a *Adapter) FindFile(ctx context.Context, id ragserver.FileID, partial authz.Partial) (*ragserver.File, error) {
	var file *ragserver.File
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := findFileQuery{
			id:      id,
			partial: partial,
		}.SQL()

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

type findFileQuery struct {
	id      ragserver.FileID
	partial authz.Partial
}

func (q findFileQuery) SQL() (string, []any) {
	query := `
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
			fs."name" as "status",
			fse."message" as "status_message",
			f."created_at",
			f."updated_at"
		from "file" f
		inner join "file_status" fs ON f."status" = fs."id"	
		inner join "file_status_evt" fse on fse."file" = f."id" and fse."status" = fs."id" 
		where f."id" = ?
	`
	args := []any{q.id}

	// Add where clauses from the partial if any
	partialClauses, partialArgs := q.partial.SQL()
	if partialClauses != "" {
		query += " and " + partialClauses

		args = append(args, partialArgs...)
	}

	return query, args
}

func scanFile(row Scannable) (*ragserver.File, error) {
	var (
		aFile         = new(ragserver.File)
		statusMessage = sql.NullString{}
	)

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
		&statusMessage,
		&aFile.CreatedAt,
		&aFile.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ragserver.ErrNotFound
		}
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	if statusMessage.Valid {
		aFile.StatusMessage = statusMessage.String
	}

	return aFile, nil
}

type listFilesForProcessing struct {
	now     ragserver.Time
	partial authz.Partial
}

func (q listFilesForProcessing) SQL() (string, []any) {
	sql := `
		update "file" set 
			"status" = (select "id" from "file_status" fs where fs."name" = ?), 
			"updated_at" = ?
		where 
			"status" = (select "id" from "file_status" fs where fs."name" = ?)
	`
	args := []any{ragserver.FileStatusProcessing, q.now, ragserver.FileStatusUploaded}

	// Add where clauses from the partial if any
	partialClauses, partialArgs := q.partial.SQL()
	if partialClauses != "" {
		sql += " and " + partialClauses

		args = append(args, partialArgs...)
	}

	sql += ` returning "id"`

	return sql, args
}

// ListFilesForProcessing lists IDs of files are in UPLOADED state and ready for PROCESSING.
// It starts with an UPDATE query to escalate transaction from read to write, this way concurrent
// transactions will not be able to select the same files even within the same sqlite DB connection.
func (a *Adapter) ListFilesForProcessing(ctx context.Context, now ragserver.Time, partial authz.Partial) ([]ragserver.FileID, error) {
	var ids []ragserver.FileID
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		// First, update files from UPLOADED to PROCESSING to lock them for this transaction
		sql, args := listFilesForProcessing{
			now:     now,
			partial: partial,
		}.SQL()

		stmt, err := tx.Prepare(sql)
		if err != nil {
			return fmt.Errorf("prepare statement failed: %w", err)
		}
		defer stmt.Close()

		rows, err := stmt.QueryContext(ctx, args...)
		if err != nil {
			return fmt.Errorf("query context failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id ragserver.FileID
			if err := rows.Scan(&id); err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
			ids = append(ids, id)
		}
		rows.Close()

		if len(ids) == 0 {
			return nil
		}

		// Append file lifecycle events for the files we just updated
		files := make([]*ragserver.File, 0, len(ids))
		for _, id := range ids {
			files = append(files, &ragserver.File{
				ID:        id,
				Status:    ragserver.FileStatusProcessing,
				CreatedAt: now,
			})
		}
		if err := execBatchInsertQuery(ctx, tx, insertFileStatusEventsQuery{files: files}); err != nil {
			return fmt.Errorf("exec batch insert query failed: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

func (a *Adapter) DeleteFiles(ctx context.Context, files ...*ragserver.File) error {
	if len(files) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execQuery(ctx, tx, deleteFileStatusEventsQuery{files: files}); err != nil {
			return fmt.Errorf("exec query failed: %w", err)
		}

		if err := execQuery(ctx, tx, deleteFilesQuery{files: files}); err != nil {
			return fmt.Errorf("exec query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type deleteFileStatusEventsQuery struct {
	files []*ragserver.File
}

func (q deleteFileStatusEventsQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	sql := `delete from "file_status_evt" where "file" in (?`
	args := make([]any, 0, len(q.files))
	args = append(args, q.files[0].ID)
	for i := range q.files[1:] {
		sql += `, ?`
		args = append(args, q.files[i+1].ID)
	}
	sql += `)`

	return sql, args
}

type deleteFilesQuery struct {
	files []*ragserver.File
}

func (q deleteFilesQuery) SQL() (string, []any) {
	if len(q.files) == 0 {
		return "", nil
	}

	sql := `delete from "file" where "id" in (?`
	args := make([]any, 0, len(q.files))
	args = append(args, q.files[0].ID)
	for i := range q.files[1:] {
		sql += `, ?`
		args = append(args, q.files[i+1].ID)
	}
	sql += `)`

	return sql, args
}
