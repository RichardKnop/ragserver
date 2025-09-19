package ragservertest

import (
	"time"

	"github.com/RichardKnop/ragserver"
)

type ScreeningOption func(*ragserver.Screening)

func WithScreeningID(id ragserver.ScreeningID) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.ID = id
	}
}

func WithScreeningAuthorID(id ragserver.AuthorID) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.AuthorID = id
	}
}

func WithScreeningStatus(status ragserver.ScreeningStatus) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.Status = status
	}
}

func WithScreeningCreated(created time.Time) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.Created = created
	}
}

func WithScreeningUpdated(updated time.Time) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.Updated = updated
	}
}

func WithScreeningFiles(files ...*ragserver.File) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.Files = files
	}
}

func WithScreeningQuestions(questions ...*ragserver.Question) ScreeningOption {
	return func(s *ragserver.Screening) {
		s.Questions = questions
	}
}

var screeningStates = []ragserver.ScreeningStatus{
	ragserver.ScreeningStatusRequested,
	ragserver.ScreeningStatusGenerating,
	ragserver.ScreeningStatusCompleted,
	ragserver.ScreeningStatusFailed,
}

func (g *DataGen) Screening(options ...ScreeningOption) *ragserver.Screening {
	g.ShuffleAnySlice(screeningStates)

	aScreening := ragserver.Screening{
		ID:       ragserver.NewScreeningID(),
		AuthorID: ragserver.NewAuthorID(),
		Status:   screeningStates[0],
		Created:  g.now,
		Updated:  g.now,
	}

	for _, o := range options {
		o(&aScreening)
	}

	return &aScreening
}
