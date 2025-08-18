package ragserver

import (
	"time"

	"github.com/neurosnap/sentences"
)

type clock func() time.Time

type ragServer struct {
	genai    GenaiAdapter
	training *sentences.Storage
	weaviate WeaviateAdapter
	pdf      PDF
	store    Store
	now      clock
}

type Option func(*ragServer)

func New(gAdapter GenaiAdapter, wvAdapter WeaviateAdapter, training *sentences.Storage, pdfAdapter PDF, storeAdapter Store, options ...Option) *ragServer {
	rs := &ragServer{
		genai:    gAdapter,
		weaviate: wvAdapter,
		training: training,
		pdf:      pdfAdapter,
		store:    storeAdapter,
		now:      time.Now,
	}

	for _, o := range options {
		o(rs)
	}

	return rs
}
