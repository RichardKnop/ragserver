package ragserver

import (
	"context"
	"io"
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

type PDF interface {
	Extract(ctx context.Context, contents io.ReadSeeker) ([]Document, error)
}

type ragServer struct {
	wvClient *weaviate.Client
	client   *genai.Client
	training *sentences.Storage
	pdf      PDF
	now      clock
}

func New(wvClient *weaviate.Client, genaiClient *genai.Client, training *sentences.Storage, pdfAdapter PDF) *ragServer {
	return &ragServer{
		wvClient: wvClient,
		client:   genaiClient,
		training: training,
		pdf:      pdfAdapter,
		now:      time.Now,
	}
}
