package ragserver

import (
	"context"
	"database/sql"
	"io"
)

type GenaiAdapter interface {
	EmbedDocuments(ctx context.Context, documents []Document) ([]Vector, error)
	EmbedContent(ctx context.Context, content string) (Vector, error)
	Generate(ctx context.Context, input string) ([]string, error)
}

type WeaviateAdapter interface {
	SaveEmbeddings(ctx context.Context, documents []Document, vectors []Vector) error
	SearchDocuments(ctx context.Context, vector Vector, fileIDs ...FileID) ([]string, error)
}

type PDF interface {
	Extract(ctx context.Context, contents io.ReadSeeker) ([]Document, error)
}

type Store interface {
	Transactional
	FileStore
}

type Transactional interface {
	Transactional(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error
}

type FileStore interface {
	SaveFile(ctx context.Context, file *File) error
	ListFiles(ctx context.Context) ([]*File, error)
	FindFile(ctx context.Context, id FileID) (*File, error)
}
