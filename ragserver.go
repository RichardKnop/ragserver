package ragserver

import (
	_ "embed"
	"errors"
	"time"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

var ErrNotFound = errors.New("not found")

//go:embed testdata/english.json
var TestEn string

type clock func() time.Time

type ragServer struct {
	extractor      Extractor
	embedder       Embedder
	retriever      Retriever
	generative     GenerativeModel
	store          Store
	now            clock
	relevantTopics RelevantTopics
}

type Option func(*ragServer)

func WithRelevantTopics(topics RelevantTopics) Option {
	return func(rs *ragServer) {
		rs.relevantTopics = topics
	}
}

func New(extractor Extractor, embedder Embedder, retriever Retriever, gm GenerativeModel, storeAdapter Store, options ...Option) *ragServer {
	rs := &ragServer{
		extractor:  extractor,
		embedder:   embedder,
		retriever:  retriever,
		generative: gm,
		store:      storeAdapter,
		now:        func() time.Time { return time.Now().UTC() },
	}

	for _, o := range options {
		o(rs)
	}

	return rs
}

func (rs *ragServer) filePpartial() authz.Partial {
	return authz.FilterBy("embedder", rs.embedder.Name()).And("retriever", rs.retriever.Name())
}

func (rs *ragServer) screeningPartial() authz.Partial {
	return authz.NilPartial
}
