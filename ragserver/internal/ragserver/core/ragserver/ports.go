package ragserver

import (
	"context"
	"database/sql"
	"io"
)

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
}
