package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite
	tempFolder string
	db         *sql.DB
	adapter    *Adapter
}

func (s *StoreTestSuite) SetupSuite() {
	d, err := os.MkdirTemp("", "ragserver-test")
	s.Require().NoError(err)
	s.tempFolder = d

	f, err := os.CreateTemp(d, "db.sqlite")
	s.Require().NoError(err)
	f.Close()

	dbConnOpts := url.Values{}
	dbConnOpts.Set("_fk", "true")
	dbConnOpts.Set("_journal", "WAL")
	dbConnOpts.Set("_timeout", "5000")

	s.db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?%s", f.Name(), dbConnOpts.Encode()))
	s.Require().NoError(err)
}

func (s *StoreTestSuite) TearDownSuite() {
	s.Require().NoError(s.db.Close())
	s.Require().NoError(os.RemoveAll(s.tempFolder))
}

func (s *StoreTestSuite) SetupTest() {
	// Migrate down and migrate up to have a clean schema
	driver, err := sqlite3.WithInstance(s.db, &sqlite3.Config{})
	s.Require().NoError(err)

	migrationsPath, err := filepath.Abs("../../db/migrations")
	s.Require().NoError(err)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"sqlite3", driver)
	s.Require().NoError(err)
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		s.Require().NoError(err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		s.Require().NoError(err)
	}
	s.adapter = New(s.db)
}

func (s *StoreTestSuite) TearDownTest() {
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}
