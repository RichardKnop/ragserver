package rest

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ai/ragserver/internal/pkg/authz"
	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type RagServer interface {
	CreateDocuments(ctx context.Context, principal authz.Principal, documents []ragserver.Document) error
	CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*ragserver.File, error)
	ListFiles(ctx context.Context, principal authz.Principal) ([]*ragserver.File, error)
	FindFile(ctx context.Context, principal authz.Principal, id ragserver.FileID) (*ragserver.File, error)
	Generate(ctx context.Context, principal authz.Principal, query ragserver.Query, fileIDs ...ragserver.FileID) ([]ragserver.Response, error)
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
	mux.HandleFunc("GET /files/", a.listFilesHandler)
	mux.HandleFunc("GET /files/{id}/", a.getFileHandler)
	mux.HandleFunc("POST /query/", a.queryHandler)
}

var staticPrincipal = authz.New(authz.ID{
	UUID: uuid.Must(uuid.FromString("b486ea88-95c4-4140-86c9-dd19f6fa879f")),
})

func (a *Adapter) principalFromRequest(req *http.Request) authz.Principal {
	// TODO - get actual principal from the request later when auth is implemented.
	// For now, we use a static hardcoded principal for testing purposes.
	return staticPrincipal
}
