package ragserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	Created       time.Time
	Updated       time.Time
}

// FileIDs returns the IDs of the files associated with the screening.
func (s *Screening) FileIDs() []FileID {
	fileIDs := make([]FileID, 0, len(s.Files))
	for _, aFile := range s.Files {
		fileIDs = append(fileIDs, aFile.ID)
	}
	return fileIDs
}

// CompleteWithStatus changes the status of a screening to a completion status,
// either ScreeningStatusCompleted or ScreeningStatusFailed.
func (s *Screening) CompleteWithStatus(newStatus ScreeningStatus, message string, updatedAt time.Time) error {
	if s.Status != ScreeningStatusGenerating {
		return fmt.Errorf("cannot change status from %s to %s", s.Status, newStatus)
	}

	s.Status = newStatus
	s.StatusMessage = message
	s.Updated = updatedAt

	return nil
}

type ScreeningFilter struct {
	Status            ScreeningStatus
	LastUpdatedBefore time.Time
	Lock              bool
}

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
	Created     time.Time
	Answered    time.Time
}

type Answer struct {
	QuestionID QuestionID
	Response   string
	Created    time.Time
}

type QuestionFilter struct {
	ScreeningID ScreeningID
}

func (rs *ragServer) CreateScreening(ctx context.Context, principal authz.Principal, params ScreeningParams) (*Screening, error) {
	if len(params.Questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}
	if len(params.FileIDs) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	files, err := rs.processedFilesFromIDs(ctx, params.FileIDs...)
	if err != nil {
		return nil, err
	}

	aScreening := &Screening{
		ID:        NewScreeningID(),
		AuthorID:  AuthorID{principal.ID().UUID},
		Status:    ScreeningStatusRequested,
		Created:   rs.now(),
		Updated:   rs.now(),
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
			Created:     rs.now(),
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
		screenings, err = rs.store.ListScreenings(ctx, ScreeningFilter{}, rs.screeningPartial(), SortParams{})
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

func (rs *ragServer) DeleteScreening(ctx context.Context, principal authz.Principal, id ScreeningID) error {
	rs.logger.Sugar().With("id", id).Info("deleting screening")

	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		aScreening, err := rs.store.FindScreening(ctx, id, rs.screeningPartial())
		if err != nil {
			return err
		}

		if aScreening.Status == ScreeningStatusRequested || aScreening.Status == ScreeningStatusGenerating {
			return fmt.Errorf("cannot delete screening in status %s", aScreening.Status)
		}

		return rs.store.DeleteScreenings(ctx, aScreening)
	}); err != nil {
		return err
	}
	return nil
}
