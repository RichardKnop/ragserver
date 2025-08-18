package ragserver

import (
	"time"

	"github.com/neurosnap/sentences"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"google.golang.org/genai"
)

const (
	generativeModelName = "gemini-2.5-flash"
	embeddingModelName  = "text-embedding-004"
)

type clock func() time.Time

type ragServer struct {
	wvClient *weaviate.Client
	client   *genai.Client
	training *sentences.Storage
	pdf      PDF
	store    Store
	now      clock
}

func New(wvClient *weaviate.Client, genaiClient *genai.Client, training *sentences.Storage, pdfAdapter PDF, storeAdapter Store) *ragServer {
	return &ragServer{
		wvClient: wvClient,
		client:   genaiClient,
		training: training,
		pdf:      pdfAdapter,
		store:    storeAdapter,
		now:      time.Now,
	}
}
