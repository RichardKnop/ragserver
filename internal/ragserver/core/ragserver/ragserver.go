package ragserver

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type clock func() time.Time

type ragServer struct {
	extractor      Extractor
	embedder       Embedder
	retriever      Retriever
	lm             LanguageModel
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

func New(extractor Extractor, embedder Embedder, retriever Retriever, lm LanguageModel, storeAdapter Store, options ...Option) *ragServer {
	rs := &ragServer{
		extractor: extractor,
		embedder:  embedder,
		retriever: retriever,
		lm:        lm,
		store:     storeAdapter,
		now:       time.Now,
	}

	for _, o := range options {
		o(rs)
	}

	return rs
}
