package rest

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

type RagServer interface {
	CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*ragserver.File, error)
	ListFiles(ctx context.Context, principal authz.Principal) ([]*ragserver.File, error)
	FindFile(ctx context.Context, principal authz.Principal, id ragserver.FileID) (*ragserver.File, error)
	ListFileDocuments(ctx context.Context, principal authz.Principal, id ragserver.FileID) ([]ragserver.Document, error)
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

const defaultTimeout = 3 * time.Second

var staticPrincipal = authz.New(authz.ID{
	UUID: uuid.Must(uuid.FromString("b486ea88-95c4-4140-86c9-dd19f6fa879f")),
})

func (a *Adapter) principalFromRequest(r *http.Request) authz.Principal {
	// TODO - get actual principal from the request later when auth is implemented.
	// For now, we use a static hardcoded principal for testing purposes.
	return staticPrincipal
}
