package ragserver

import (
	"errors"
	"time"

	"github.com/neurosnap/sentences"
)

var ErrNotFound = errors.New("not found")

type clock func() time.Time

type ragServer struct {
	genai          GenaiAdapter
	training       *sentences.Storage
	weaviate       WeaviateAdapter
	extract        ExtractAdapter
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

func New(genaiAdapter GenaiAdapter, wvAdapter WeaviateAdapter, training *sentences.Storage, extractAdapter ExtractAdapter, storeAdapter Store, options ...Option) *ragServer {
	rs := &ragServer{
		genai:    genaiAdapter,
		weaviate: wvAdapter,
		training: training,
		extract:  extractAdapter,
		store:    storeAdapter,
		now:      time.Now,
	}

	for _, o := range options {
		o(rs)
	}

	return rs
}
