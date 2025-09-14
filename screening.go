package ragserver

import (
	"context"
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
}

type QuestionFilter struct {
	ScreeningID ScreeningID
}

func (rs *ragServer) CreateScreening(ctx context.Context, principal authz.Principal, params ScreeningParams) (*Screening, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rs *ragServer) ListScreenings(ctx context.Context, principal authz.Principal) ([]*Screening, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rs *ragServer) FindScreening(ctx context.Context, principal authz.Principal, id ScreeningID) (*Screening, error) {
	return nil, fmt.Errorf("not implemented")
}
