package rest

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type RagServer interface {
	CreateDocuments(ctx context.Context, documents []ragserver.Document) error
	CreateFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*ragserver.File, error)
	Generate(ctx context.Context, queryy ragserver.Query) ([]ragserver.Response, error)
}

type Adapter struct {
	ragServer RagServer
}

func New(ragServer RagServer) *Adapter {
	return &Adapter{
		ragServer: ragServer,
	}
}

func (a *Adapter) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /documents/", a.createDocumentsHandler)
	mux.HandleFunc("POST /files/", a.uploadFileHandler)
	mux.HandleFunc("POST /query/", a.queryHandler)
}
