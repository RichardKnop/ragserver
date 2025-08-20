package pdf

import (
	"math"

	"github.com/neurosnap/sentences"
)

type Adapter struct {
	extractor *extractor
	training  *sentences.Storage
}

type Option func(*Adapter)

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
	}

	for _, o := range options {
		o(a)
	}

	return a
}
