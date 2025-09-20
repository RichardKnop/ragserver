package ragserver

import (
	_ "embed"
	"errors"
	"time"

	"go.uber.org/zap"

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
	filestorage    FileStorage
	now            clock
	relevantTopics RelevantTopics
	logger         *zap.Logger
}

type Option func(*ragServer)

func WithRelevantTopics(topics RelevantTopics) Option {
	return func(rs *ragServer) {
		rs.relevantTopics = topics
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(rs *ragServer) {
		rs.logger = logger
	}
}

func New(extractor Extractor, embedder Embedder, retriever Retriever, gm GenerativeModel, storeAdapter Store, fileStorage FileStorage, options ...Option) *ragServer {
	rs := &ragServer{
		extractor:   extractor,
		embedder:    embedder,
		retriever:   retriever,
		generative:  gm,
		store:       storeAdapter,
		filestorage: fileStorage,
		now:         func() time.Time { return time.Now().UTC() },
		logger:      zap.NewNop(),
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
