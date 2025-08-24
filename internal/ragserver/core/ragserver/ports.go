package ragserver

import (
	"context"
	"database/sql"
	"io"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
)

// Extractor extracts documents from various contents, optionally limited by relevant topics.
type Extractor interface {
	Extract(ctx context.Context, contents io.ReadSeeker, topics RelevantTopics) ([]Document, error)
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
	ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error)
	SearchDocuments(ctx context.Context, vector Vector, limit int, fileIDs ...FileID) ([]Document, error)
}

// LanguageModel uses generative AI to generate responses based on a query and relevant documents.
type LanguageModel interface {
	Generate(ctx context.Context, query Query, documents []Document) ([]Response, error)
}

type Store interface {
	Transactional
	FileStore
}

type Transactional interface {
	Transactional(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error
}

type FileStore interface {
	SaveFiles(ctx context.Context, file ...*File) error
	ListFiles(ctx context.Context, filter FileFilter, partial authz.Partial) ([]*File, error)
	FindFile(ctx context.Context, id FileID, partial authz.Partial) (*File, error)
}
