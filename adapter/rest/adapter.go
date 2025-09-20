package rest

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
)

type RagServer interface {
	CreateFile(ctx context.Context, principal authz.Principal, file io.ReadSeeker, header *multipart.FileHeader) (*ragserver.File, error)
	ListFiles(ctx context.Context, principal authz.Principal) ([]*ragserver.File, error)
	FindFile(ctx context.Context, principal authz.Principal, id ragserver.FileID) (*ragserver.File, error)
	ListFileDocuments(ctx context.Context, principal authz.Principal, id ragserver.FileID) ([]ragserver.Document, error)
	DeleteFile(ctx context.Context, principal authz.Principal, id ragserver.FileID) error
	CreateScreening(ctx context.Context, principal authz.Principal, params ragserver.ScreeningParams) (*ragserver.Screening, error)
	ListScreenings(ctx context.Context, principal authz.Principal) ([]*ragserver.Screening, error)
	FindScreening(ctx context.Context, principal authz.Principal, id ragserver.ScreeningID) (*ragserver.Screening, error)
	DeleteScreening(ctx context.Context, principal authz.Principal, id ragserver.ScreeningID) error
}

type Adapter struct {
	ragServer RagServer
	logger    *zap.Logger
}

type Option func(*Adapter)

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

func New(ragServer RagServer, options ...Option) *Adapter {
	a := &Adapter{
		ragServer: ragServer,
		logger:    zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	return a
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
