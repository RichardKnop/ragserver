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

func (a *Adapter) SaveScreenings(ctx context.Context, screenings ...*ragserver.Screening) error {
	if len(screenings) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execQueryCheckRowsAffected(ctx, tx, insertScreeningsQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec insert screenings query failed: %w", err)
		}

		if err := execQueryCheckRowsAffected(ctx, tx, insertScreeningStatusEventsQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec insert screening status events query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *Adapter) SaveScreeningFiles(ctx context.Context, screenings ...*ragserver.Screening) error {
	if len(screenings) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		for _, aScreening := range screenings {
			if len(aScreening.Files) == 0 {
				continue
			}

			if err := execQueryCheckRowsAffected(ctx, tx, insertScreeningFilesQuery{aScreening}); err != nil {
				return fmt.Errorf("exec insert screening files query failed: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *Adapter) SaveScreeningQuestions(ctx context.Context, screenings ...*ragserver.Screening) error {
	if len(screenings) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		for _, aScreening := range screenings {
			if len(aScreening.Questions) == 0 {
				continue
			}

			if err := execQueryCheckRowsAffected(ctx, tx, insertScreeningQuestionsQuery{aScreening}); err != nil {
				return fmt.Errorf("exec insert screening questions query failed: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type insertScreeningsQuery struct {
	screenings []*ragserver.Screening
}

func (q insertScreeningsQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `
		insert into "ragserver"."screening" (
			"id",
			"author",
			"status",
			"created",
			"updated"
		)
		values (?, ?, (select "id" from "ragserver"."screening_status" fs where fs."name" = ?), ?, ?)
	`
	args := make([]any, 0, len(q.screenings)*5)
	args = append(
		args,
		q.screenings[0].ID,
		q.screenings[0].AuthorID,
		q.screenings[0].Status,
		q.screenings[0].Created,
		q.screenings[0].Updated,
	)
	for i := range q.screenings[1:] {
		query += `, (?, ?, (select "id" from "ragserver"."screening_status" fs where fs."name" = ?), ?, ?)`
		args = append(
			args,
			q.screenings[i+1].ID,
			q.screenings[i+1].AuthorID,
			q.screenings[i+1].Status,
			q.screenings[i+1].Created,
			q.screenings[i+1].Updated,
		)
	}
	query += `
		on conflict("id") do update set
			"author"=excluded."author",
			"status"=excluded."status",
			"updated"=excluded."updated"
	`

	return toPostgresParams(query), args
}

type insertScreeningStatusEventsQuery struct {
	screenings []*ragserver.Screening
}

func (q insertScreeningStatusEventsQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `
		insert into "ragserver"."screening_status_evt" (
			"screening", 
			"status",
			"message",
			"created"
		)
		values (?, (select "id" from "ragserver"."screening_status" fs where fs."name" = ?), ?, ?)
	`
	args := make([]any, 0, len(q.screenings)*4)
	args = append(
		args,
		q.screenings[0].ID,
		q.screenings[0].Status,
		sql.NullString{String: q.screenings[0].StatusMessage, Valid: q.screenings[0].StatusMessage != ""},
		q.screenings[0].Created,
	)
	for i := range q.screenings[1:] {
		query += `, (?, (select "id" from "ragserver"."screening_status" fs where fs."name" = ?), ?, ?)`
		args = append(
			args,
			q.screenings[i+1].ID,
			q.screenings[i+1].Status,
			sql.NullString{String: q.screenings[i+1].StatusMessage, Valid: q.screenings[i+1].StatusMessage != ""},
			q.screenings[i+1].Created,
		)
	}

	return toPostgresParams(query), args
}

type insertScreeningFilesQuery struct {
	*ragserver.Screening
}

func (q insertScreeningFilesQuery) SQL() (string, []any) {
	if len(q.Files) == 0 {
		return "", nil
	}

	query := `
		insert into "ragserver"."screening_file" (
			"screening", 
			"file",
			"order"
		)
		values (?, ?, ?)
	`
	args := make([]any, 0, len(q.Files)*2)
	args = append(
		args,
		q.ID,
		q.Files[0].ID,
		0, // order
	)
	for i := range q.Files[1:] {
		query += `, (?, ?, ?)`
		args = append(
			args,
			q.ID,
			q.Files[i+1].ID,
			i+1, // order
		)
	}

	return toPostgresParams(query), args
}

type insertScreeningQuestionsQuery struct {
	*ragserver.Screening
}

func (q insertScreeningQuestionsQuery) SQL() (string, []any) {
	if len(q.Questions) == 0 {
		return "", nil
	}

	query := `
		insert into "ragserver"."question" (
			"id",
			"author",
			"type", 
			"content",
			"screening",
			"order",
			"created"
		)
		values (?, ?, (select "id" from "ragserver"."question_type" fs where fs."name" = ?), ?, ?, ?, ?)
	`
	args := make([]any, 0, len(q.Questions)*8)
	args = append(
		args,
		q.Questions[0].ID,
		q.Questions[0].AuthorID,
		q.Questions[0].Type,
		q.Questions[0].Content,
		q.Questions[0].ScreeningID,
		0, // order
		q.Questions[0].Created,
	)
	for i := range q.Questions[1:] {
		query += `, (?, ?, (select "id" from "ragserver"."question_type" fs where fs."name" = ?), ?, ?, ?, ?)`
		args = append(
			args,
			q.Questions[i+1].ID,
			q.Questions[i+1].AuthorID,
			q.Questions[i+1].Type,
			q.Questions[i+1].Content,
			q.Questions[i+1].ScreeningID,
			i+1, // order
			q.Questions[i+1].Created,
		)
	}

	return toPostgresParams(query), args
}

var (
	validScreeningSortFields = []string{
		`s."created"`,
	}
	defaultScreeningSortParams = ragserver.SortParams{
		By: `s."created"`, Order: ragserver.SortOrderDesc,
		Limit: 100,
	}
)

func (a *Adapter) ListScreenings(ctx context.Context, filter ragserver.ScreeningFilter, partial authz.Partial, params ragserver.SortParams) ([]*ragserver.Screening, error) {
	var screenings []*ragserver.Screening

	// Validate params
	if !params.Empty() && !params.Valid(validScreeningSortFields) {
		return nil, fmt.Errorf("invalid sort params: %v", params)
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		sql, args := selectScreeningsQuery{
			filter:  filter,
			partial: partial,
			params:  params,
		}.SQL()

		rows, err := tx.QueryContext(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("select screenings query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var aScreening = new(ragserver.Screening)
			aScreening, err = scanScreening(rows)
			if err != nil {
				return err
			}
			screenings = append(screenings, aScreening)
		}

		rows.Close()

		for _, aScreening := range screenings {
			// Select files
			sql, args = selectFilesQuery{
				filter:  ragserver.FileFilter{ScreeningID: aScreening.ID},
				partial: partial,
				params:  ragserver.SortParams{By: `sf."order"`, Order: ragserver.SortOrderAsc},
			}.SQL()

			rows, err = tx.QueryContext(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("select screening files query failed: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var aFile = new(ragserver.File)
				aFile, err = scanFile(rows)
				if err != nil {
					return err
				}
				aScreening.Files = append(aScreening.Files, aFile)
			}

			rows.Close()

			// Select questions
			sql, args = selectQuestionsQuery{
				filter:  ragserver.QuestionFilter{ScreeningID: aScreening.ID},
				partial: partial,
				params:  ragserver.SortParams{By: `q."order"`, Order: ragserver.SortOrderAsc},
			}.SQL()

			rows, err = tx.QueryContext(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("select screening questions query failed: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var aQuestion = new(ragserver.Question)
				aQuestion, err = scanQuestion(rows)
				if err != nil {
					return err
				}
				aScreening.Questions = append(aScreening.Questions, aQuestion)
			}

			rows.Close()

			// Select answers
			if len(aScreening.Questions) > 0 {
				sql, args = selectAnswersQuery{questions: aScreening.Questions}.SQL()

				rows, err = tx.QueryContext(ctx, sql, args...)
				if err != nil {
					return fmt.Errorf("select screening answers query failed: %w", err)
				}
				defer rows.Close()

				for rows.Next() {
					var anAnswer ragserver.Answer
					anAnswer, err = scanAnswer(rows)
					if err != nil {
						return err
					}
					aScreening.Answers = append(aScreening.Answers, anAnswer)
				}

				rows.Close()
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return screenings, nil
}

type selectScreeningsQuery struct {
	filter  ragserver.ScreeningFilter
	partial authz.Partial
	params  ragserver.SortParams
}

func (q selectScreeningsQuery) SQL() (string, []any) {
	query := `
		select 
			s."id",
			s."author",
			ss."name" as "status",
			sse."message" as "status_message",
			s."created",
			s."updated"
		from "ragserver"."screening" s
		inner join "screening_status" ss on s."status" = ss."id"
		inner join "screening_status_evt" sse on sse."screening" = s."id" and sse."status" = ss."id"
	`
	args := []any{}

	// Add where clauses from the filter and/or partial if any
	where, whereArgs := screeningFilterClauses(q.filter)
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

	// Add order by clause and/or limit if any
	if q.params.Empty() {
		q.params = defaultScreeningSortParams
	}
	query += q.params.SQL()

	if q.filter.Lock {
		query += " for update skip locked"
	}

	return toPostgresParams(query), args
}

func screeningFilterClauses(filter ragserver.ScreeningFilter) (string, []any) {
	var (
		clauses = []string{}
		args    = []any{}
	)

	if filter.Status != "" {
		clauses = append(clauses, `ss."name" = ?`)
		args = append(args, filter.Status)
	}

	if !filter.LastUpdatedBefore.IsZero() {
		clauses = append(clauses, `s."updated" < ?`)
		args = append(args, filter.LastUpdatedBefore)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " and "), args
}

func (a *Adapter) FindScreening(ctx context.Context, id ragserver.ScreeningID, partial authz.Partial) (*ragserver.Screening, error) {
	var aScreening *ragserver.Screening
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		query, args := findScreeningQuery{
			id:      id,
			partial: partial,
		}.SQL()

		stmt, err := tx.Prepare(query)
		if err != nil {
			return fmt.Errorf("prepare find screening statement failed: %w", err)
		}
		defer stmt.Close()

		row := stmt.QueryRowContext(ctx, args...)
		aScreening, err = scanScreening(row)
		if err != nil {
			return err
		}

		// Select files
		sql, args := selectFilesQuery{
			filter:  ragserver.FileFilter{ScreeningID: aScreening.ID},
			partial: partial,
			params:  ragserver.SortParams{By: `sf."order"`, Order: ragserver.SortOrderAsc},
		}.SQL()

		rows, err := tx.QueryContext(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("select screening files query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var aFile = new(ragserver.File)
			aFile, err = scanFile(rows)
			if err != nil {
				return err
			}
			aScreening.Files = append(aScreening.Files, aFile)
		}

		rows.Close()

		// Select questions
		sql, args = selectQuestionsQuery{
			filter:  ragserver.QuestionFilter{ScreeningID: aScreening.ID},
			partial: partial,
			params:  ragserver.SortParams{By: `q."order"`, Order: ragserver.SortOrderAsc},
		}.SQL()

		rows, err = tx.QueryContext(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("select screening questions query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var aQuestion = new(ragserver.Question)
			aQuestion, err = scanQuestion(rows)
			if err != nil {
				return err
			}
			aScreening.Questions = append(aScreening.Questions, aQuestion)
		}

		rows.Close()

		if len(aScreening.Questions) > 0 {
			// Select answers
			sql, args = selectAnswersQuery{questions: aScreening.Questions}.SQL()

			rows, err = tx.QueryContext(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("select screening answers query failed: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var anAnswer ragserver.Answer
				anAnswer, err = scanAnswer(rows)
				if err != nil {
					return err
				}
				aScreening.Answers = append(aScreening.Answers, anAnswer)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return aScreening, nil
}

type findScreeningQuery struct {
	id      ragserver.ScreeningID
	partial authz.Partial
}

func (q findScreeningQuery) SQL() (string, []any) {
	query := `
		select 
			s."id",
			s."author",
			ss."name" as "status",
			sse."message" as "status_message",
			s."created",
			s."updated"
		from "ragserver"."screening" s
		inner join "screening_status" ss on s."status" = ss."id"	
		inner join "screening_status_evt" sse on sse."screening" = s."id" and sse."status" = ss."id" 
		where s."id" = ?
	`
	args := []any{q.id}

	// Add where clauses from the partial if any
	partialClauses, partialArgs := q.partial.SQL()
	if partialClauses != "" {
		query += " and " + partialClauses

		args = append(args, partialArgs...)
	}

	return toPostgresParams(query), args
}

func scanScreening(row Scannable) (*ragserver.Screening, error) {
	var (
		aScreening    = new(ragserver.Screening)
		statusMessage = sql.NullString{}
		created       sql.NullTime
		updated       sql.NullTime
	)

	if err := row.Scan(
		&aScreening.ID,
		&aScreening.AuthorID,
		&aScreening.Status,
		&statusMessage,
		&created,
		&updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ragserver.ErrNotFound
		}
		return nil, fmt.Errorf("scan screening failed: %w", err)
	}

	if statusMessage.Valid {
		aScreening.StatusMessage = statusMessage.String
	}

	aScreening.Created = created.Time.UTC()
	aScreening.Updated = updated.Time.UTC()

	return aScreening, nil
}

type selectQuestionsQuery struct {
	filter  ragserver.QuestionFilter
	partial authz.Partial
	params  ragserver.SortParams
}

func (q selectQuestionsQuery) SQL() (string, []any) {
	query := `
		select 
			q."id",
			q."author",
			qt."name" as "type",
			q."content",
			q."screening",
			q."created",
			q."answered"
		from "ragserver"."question" q
		inner join "question_type" qt on q."type" = qt."id"
	`
	args := []any{}

	// Add where clauses from the filter and/or partial if any
	where, whereArgs := questionFilterClauses(q.filter)
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

	if !q.params.Empty() {
		query += q.params.SQL()
	}

	return toPostgresParams(query), args
}

func questionFilterClauses(filter ragserver.QuestionFilter) (string, []any) {
	var (
		clauses = []string{}
		args    = []any{}
	)

	if !filter.ScreeningID.UUID.IsNil() {
		clauses = append(clauses, `q."screening" = ?`)
		args = append(args, filter.ScreeningID)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
}

func scanQuestion(row Scannable) (*ragserver.Question, error) {
	var (
		aQuestion = new(ragserver.Question)
		created   sql.NullTime
		answered  sql.NullTime
	)

	if err := row.Scan(
		&aQuestion.ID,
		&aQuestion.AuthorID,
		&aQuestion.Type,
		&aQuestion.Content,
		&aQuestion.ScreeningID,
		&created,
		&answered,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ragserver.ErrNotFound
		}
		return nil, fmt.Errorf("scan question failed: %w", err)
	}

	aQuestion.Created = created.Time.UTC()
	if answered.Valid {
		aQuestion.Answered = answered.Time.UTC()
	}

	return aQuestion, nil
}

type selectAnswersQuery struct {
	questions []*ragserver.Question
}

func (q selectAnswersQuery) SQL() (string, []any) {
	if len(q.questions) == 0 {
		return "", nil
	}

	query := `
		select 
			a."question",
			a."response",
			a."created"
		from "ragserver"."answer" a
		where a."question" in (?
	`
	args := make([]any, 0, len(q.questions))
	args = append(args, q.questions[0].ID)
	for i := range q.questions[1:] {
		query += `, ?`
		args = append(args, q.questions[i+1].ID)
	}
	query += `)`

	return toPostgresParams(query), args
}

func scanAnswer(row Scannable) (ragserver.Answer, error) {
	var (
		anAnswer = ragserver.Answer{}
		created  sql.NullTime
	)

	if err := row.Scan(
		&anAnswer.QuestionID,
		&anAnswer.Response,
		&created,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ragserver.Answer{}, ragserver.ErrNotFound
		}
		return ragserver.Answer{}, fmt.Errorf("scan answer failed: %w", err)
	}

	anAnswer.Created = created.Time.UTC()

	return anAnswer, nil
}

func (a *Adapter) SaveAnswer(ctx context.Context, answer ragserver.Answer) error {
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execQuery(ctx, tx, insertAnswerQuery{answer}); err != nil {
			return fmt.Errorf("exec insert answer query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type insertAnswerQuery struct {
	ragserver.Answer
}

func (q insertAnswerQuery) SQL() (string, []any) {
	query := `
		insert into "ragserver"."answer" ("question", "response", "created")
		values (?, ?, ?)
	`
	args := []any{q.QuestionID, q.Response, q.Created}

	return toPostgresParams(query), args
}

func (a *Adapter) DeleteScreenings(ctx context.Context, screenings ...*ragserver.Screening) error {
	if len(screenings) < 1 {
		return nil
	}

	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execQuery(ctx, tx, deleteScreeningFilesQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec delete screening files query failed: %w", err)
		}

		if err := execQuery(ctx, tx, deleteScreeningAnswersQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec delete screening answers query failed: %w", err)
		}

		if err := execQuery(ctx, tx, deleteScreeningQuestionsQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec delete screening questions query failed: %w", err)
		}

		if err := execQuery(ctx, tx, deleteScreeningStatusEventsQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec delete screening status events query failed: %w", err)
		}

		if err := execQuery(ctx, tx, deleteScreeningsQuery{screenings: screenings}); err != nil {
			return fmt.Errorf("exec delete screenings query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type deleteScreeningFilesQuery struct {
	screenings []*ragserver.Screening
}

func (q deleteScreeningFilesQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `delete from "ragserver"."screening_file" where "screening" in (?`
	args := make([]any, 0, len(q.screenings))
	args = append(args, q.screenings[0].ID)
	for i := range q.screenings[1:] {
		query += `, ?`
		args = append(args, q.screenings[i+1].ID)
	}
	query += `)`

	return toPostgresParams(query), args
}

type deleteScreeningAnswersQuery struct {
	screenings []*ragserver.Screening
}

func (q deleteScreeningAnswersQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `delete from "ragserver"."answer" where "question" in (select "id" from "ragserver"."question" where "screening" in (?`
	args := make([]any, 0, len(q.screenings))
	args = append(args, q.screenings[0].ID)
	for i := range q.screenings[1:] {
		query += `, ?`
		args = append(args, q.screenings[i+1].ID)
	}
	query += `))`

	return toPostgresParams(query), args
}

type deleteScreeningQuestionsQuery struct {
	screenings []*ragserver.Screening
}

func (q deleteScreeningQuestionsQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `delete from "ragserver"."question" where "screening" in (?`
	args := make([]any, 0, len(q.screenings))
	args = append(args, q.screenings[0].ID)
	for i := range q.screenings[1:] {
		query += `, ?`
		args = append(args, q.screenings[i+1].ID)
	}
	query += `)`

	return toPostgresParams(query), args
}

type deleteScreeningStatusEventsQuery struct {
	screenings []*ragserver.Screening
}

func (q deleteScreeningStatusEventsQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `delete from "ragserver"."screening_status_evt" where "screening" in (?`
	args := make([]any, 0, len(q.screenings))
	args = append(args, q.screenings[0].ID)
	for i := range q.screenings[1:] {
		query += `, ?`
		args = append(args, q.screenings[i+1].ID)
	}
	query += `)`

	return toPostgresParams(query), args
}

type deleteScreeningsQuery struct {
	screenings []*ragserver.Screening
}

func (q deleteScreeningsQuery) SQL() (string, []any) {
	if len(q.screenings) == 0 {
		return "", nil
	}

	query := `delete from "ragserver"."screening" where "id" in (?`
	args := make([]any, 0, len(q.screenings))
	args = append(args, q.screenings[0].ID)
	for i := range q.screenings[1:] {
		query += `, ?`
		args = append(args, q.screenings[i+1].ID)
	}
	query += `)`

	return toPostgresParams(query), args
}
