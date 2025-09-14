package rest

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
)

type RagServer interface {
	CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*ragserver.File, error)
	ListFiles(ctx context.Context, principal authz.Principal) ([]*ragserver.File, error)
	FindFile(ctx context.Context, principal authz.Principal, id ragserver.FileID) (*ragserver.File, error)
	ListFileDocuments(ctx context.Context, principal authz.Principal, id ragserver.FileID) ([]ragserver.Document, error)
	Generate(ctx context.Context, principal authz.Principal, question ragserver.Question, fileIDs ...ragserver.FileID) ([]ragserver.Response, error)
	CreateScreening(ctx context.Context, principal authz.Principal, params ragserver.ScreeningParams) (*ragserver.Screening, error)
	ListScreenings(ctx context.Context, principal authz.Principal) ([]*ragserver.Screening, error)
	FindScreening(ctx context.Context, principal authz.Principal, id ragserver.ScreeningID) (*ragserver.Screening, error)
}

type Adapter struct {
	ragServer RagServer
}

func New(ragServer RagServer) *Adapter {
	return &Adapter{
		ragServer: ragServer,
	}
}

const (
	defaultTimeout = 3 * time.Second
)

var (
	principalID     = authz.ID{UUID: uuid.Must(uuid.FromString("b486ea88-95c4-4140-86c9-dd19f6fa879f"))}
	staticPrincipal = authz.New(principalID, "static-user")
)

func (a *Adapter) principalFromRequest(r *http.Request) authz.Principal {
	// TODO - get actual principal from the request later when auth is implemented.
	// For now, we use a static hardcoded principal for testing purposes.
	return staticPrincipal
}
