package store

import (
	"database/sql"
)

type Adapter struct {
	db *sql.DB
}

type Option func(*Adapter)

func New(db *sql.DB, options ...Option) *Adapter {
	a := &Adapter{
		db: db,
	}

	for _, o := range options {
		o(a)
	}

	return a
}
