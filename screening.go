package ragserver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

type ScreeningStatus string

const (
	ScreeningStatusRequested  ScreeningStatus = "REQUESTED"
	ScreeningStatusGenerating ScreeningStatus = "GENERATING"
	ScreeningStatusCompleted  ScreeningStatus = "COMPLETED"
	ScreeningStatusFailed     ScreeningStatus = "FAILED"
)

type ScreeningID struct{ uuid.UUID }

func NewScreeningID() ScreeningID {
	return ScreeningID{uuid.Must(uuid.NewV4())}
}

type ScreeningParams struct {
	FileIDs   []FileID
	Questions []Question
}

type Screening struct {
	ID            ScreeningID
	AuthorID      AuthorID
	Files         []*File
	Questions     []*Question
	Answers       []Answer
	Status        ScreeningStatus
	StatusMessage string
	Created       Time
	Updated       Time
}

type ScreeningFilter struct{}

type QuestionType string

const (
	QuestionTypeText    QuestionType = "TEXT"
	QuestionTypeBoolean QuestionType = "BOOLEAN"
	QuestionTypeMetric  QuestionType = "METRIC"
)

type QuestionID struct{ uuid.UUID }

func NewQuestionID() QuestionID {
	return QuestionID{uuid.Must(uuid.NewV4())}
}

type Question struct {
	ID          QuestionID
	AuthorID    AuthorID
	ScreeningID ScreeningID
	Type        QuestionType
	Content     string
	Created     Time
	Answered    Time
}

type Answer struct {
	QuestionID QuestionID
	Response   string
	Created    Time
}

type QuestionFilter struct {
	ScreeningID ScreeningID
}

func (rs *ragServer) CreateScreening(ctx context.Context, principal authz.Principal, params ScreeningParams) (*Screening, error) {
	files, err := rs.processedFilesFromIDs(ctx, params.FileIDs...)
	if err != nil {
		return nil, err
	}

	aScreening := &Screening{
		ID:        NewScreeningID(),
		AuthorID:  AuthorID{principal.ID().UUID},
		Status:    ScreeningStatusRequested,
		Created:   Time{rs.now()},
		Updated:   Time{rs.now()},
		Files:     files,
		Questions: make([]*Question, 0, len(params.Questions)),
	}

	for _, aQuestion := range params.Questions {
		aScreening.Questions = append(aScreening.Questions, &Question{
			ID:          NewQuestionID(),
			AuthorID:    AuthorID{principal.ID().UUID},
			ScreeningID: aScreening.ID,
			Type:        aQuestion.Type,
			Content:     aQuestion.Content,
			Created:     Time{rs.now()},
		})
	}

	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := rs.store.SavePrincipal(ctx, principal); err != nil {
			return fmt.Errorf("error saving principal: %w", err)
		}

		if err := rs.store.SaveScreenings(ctx, aScreening); err != nil {
			return fmt.Errorf("error saving screening: %w", err)
		}

		if err := rs.store.SaveScreeningFiles(ctx, aScreening); err != nil {
			return fmt.Errorf("error saving screening files: %w", err)
		}

		if err := rs.store.SaveScreeningQuestions(ctx, aScreening); err != nil {
			return fmt.Errorf("error saving screening questions: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error saving screening: %v", err)
	}

	return aScreening, nil
}

func (rs *ragServer) ListScreenings(ctx context.Context, principal authz.Principal) ([]*Screening, error) {
	var screenings []*Screening
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		screenings, err = rs.store.ListScreenings(ctx, ScreeningFilter{}, rs.screeningPartial())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return screenings, nil
}

func (rs *ragServer) FindScreening(ctx context.Context, principal authz.Principal, id ScreeningID) (*Screening, error) {
	var aScreening *Screening
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		aScreening, err = rs.store.FindScreening(ctx, id, rs.screeningPartial())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return aScreening, nil
}
