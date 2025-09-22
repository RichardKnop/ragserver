package ragserver

import (
	"context"
	"database/sql"
	"io"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

// Extractor extracts documents from various contents, optionally limited by relevant topics.
type Extractor interface {
	Extract(ctx context.Context, fileName string, contents io.ReadSeeker, topics RelevantTopics) ([]Document, error)
}

// Embedder encodes document passages as vectors
type Embedder interface {
	Name() string
	EmbedDocuments(ctx context.Context, documents []Document) ([]Vector, error)
	EmbedContent(ctx context.Context, content string) (Vector, error)
}

// Retriever that runs a question through the embeddings model and returns any encoded documents near the embedded question.
type Retriever interface {
	Name() string
	SaveDocuments(ctx context.Context, documents []Document, vectors []Vector) error
	ListFileDocuments(ctx context.Context, id FileID, limit int) ([]Document, error)
	SearchDocuments(ctx context.Context, filter DocumentFilter, limit int) ([]Document, error)
	DeleteFileDocuments(ctx context.Context, id FileID) error
}

// GenerativeModel uses generative AI to generate responses based on a query and relevant documents.
type GenerativeModel interface {
	Generate(ctx context.Context, question Question, documents []Document) ([]Response, error)
}

type Store interface {
	Transactional
	FileStore
	ScreeningStgore
}

type Transactional interface {
	Transactional(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error
}

type FileStore interface {
	SavePrincipal(ctx context.Context, principal authz.Principal) error
	SaveFiles(ctx context.Context, file ...*File) error
	ListFiles(ctx context.Context, filter FileFilter, partial authz.Partial, params SortParams) ([]*File, error)
	FindFile(ctx context.Context, id FileID, partial authz.Partial) (*File, error)
	DeleteFiles(ctx context.Context, files ...*File) error
}

type ScreeningStgore interface {
	SaveScreenings(ctx context.Context, screenings ...*Screening) error
	SaveScreeningFiles(ctx context.Context, screenings ...*Screening) error
	SaveScreeningQuestions(ctx context.Context, screenings ...*Screening) error
	ListScreenings(ctx context.Context, filter ScreeningFilter, partial authz.Partial, params SortParams) ([]*Screening, error)
	FindScreening(ctx context.Context, id ScreeningID, partial authz.Partial) (*Screening, error)
	DeleteScreenings(ctx context.Context, screenings ...*Screening) error
	SaveAnswer(ctx context.Context, answer Answer) error
}

type FileStorage interface {
	NewTempFile() (TempFile, error)
	DeleteTempFile(name string) error
	Write(filename string, data io.Reader) error
	Exists(filename string) (bool, error)
	Read(filename string) (io.ReadSeekCloser, error)
	Delete(filename string) error
}
