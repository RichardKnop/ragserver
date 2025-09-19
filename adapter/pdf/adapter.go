package pdf

import (
	"math"

	"go.uber.org/zap"

	"github.com/neurosnap/sentences"
)

type Adapter struct {
	extractor *extractor
	training  *sentences.Storage
	logger    *zap.Logger
}

type Option func(*Adapter)

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

func New(training *sentences.Storage, options ...Option) *Adapter {
	a := &Adapter{
		extractor: &extractor{
			pageMin:         1,
			pageMax:         1000,
			xRangeMin:       math.Inf(-1),
			xRangeMax:       math.Inf(1),
			showPageNumbers: false,
		},
		training: training,
		logger:   zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	return a
}
