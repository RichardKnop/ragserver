package ragservertest

import (
	"time"

	"github.com/RichardKnop/ragserver"
)

type QuestionOption func(*ragserver.Question)

func WithQuestionAuthorID(id ragserver.AuthorID) QuestionOption {
	return func(q *ragserver.Question) {
		q.AuthorID = id
	}
}

func WithQuestionScreeningID(id ragserver.ScreeningID) QuestionOption {
	return func(q *ragserver.Question) {
		q.ScreeningID = id
	}
}

func WithQuestionType(qtype ragserver.QuestionType) QuestionOption {
	return func(q *ragserver.Question) {
		q.Type = qtype
	}
}

func WithQuestionContent(content string) QuestionOption {
	return func(q *ragserver.Question) {
		q.Content = content
	}
}

func WithQuestionCreated(created time.Time) QuestionOption {
	return func(q *ragserver.Question) {
		q.Created = created
	}
}

var questionTypes = []ragserver.QuestionType{
	ragserver.QuestionTypeText,
	ragserver.QuestionTypeBoolean,
	ragserver.QuestionTypeMetric,
}

func (g *DataGen) Question(options ...QuestionOption) *ragserver.Question {
	g.ShuffleAnySlice(questionTypes)

	aQuestion := ragserver.Question{
		ID:       ragserver.NewQuestionID(),
		AuthorID: ragserver.NewAuthorID(),
		Type:     questionTypes[0],
		Created:  g.now,
	}

	for _, o := range options {
		o(&aQuestion)
	}

	return &aQuestion
}
