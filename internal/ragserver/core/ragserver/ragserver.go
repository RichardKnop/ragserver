package ragserver

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type clock func() time.Time

type ragServer struct {
	lm             LanguageModel
	retriever      Retriever
	embedder       Embedder
	extractor      Extractor
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

func New(lm LanguageModel, embedder Embedder, retriever Retriever, extractor Extractor, storeAdapter Store, options ...Option) *ragServer {
	rs := &ragServer{
		lm:        lm,
		retriever: retriever,
		embedder:  embedder,
		extractor: extractor,
		store:     storeAdapter,
		now:       time.Now,
	}

	for _, o := range options {
		o(rs)
	}

	return rs
}
