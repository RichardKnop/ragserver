package ragserver

import (
	"context"
	"database/sql"
	"io"
)

// LanguageModel receives the records from the retriever together with the question and returns an answer.
type LanguageModel interface {
	Generate(ctx context.Context, query Query, documents []Document) ([]Response, error)
}

// Embedder encodes document passages as vectors
type Embedder interface {
	EmbedDocuments(ctx context.Context, documents []Document) ([]Vector, error)
	EmbedContent(ctx context.Context, content string) (Vector, error)
}

// Retriever that runs a question through the embeddings model and returns any encoded documents near the embedded question.
type Retriever interface {
	SaveDocuments(ctx context.Context, documents []Document, vectors []Vector) error
	SearchDocuments(ctx context.Context, vector Vector, fileIDs ...FileID) ([]Document, error)
}

// Extractor extracts documents from various contents, optionally limited by relevant topics.
type Extractor interface {
	Extract(ctx context.Context, contents io.ReadSeeker, topics RelevantTopics) ([]Document, error)
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
