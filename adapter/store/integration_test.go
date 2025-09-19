package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
)

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite
	container *dockertest.Resource
	db        *sql.DB
	adapter   *Adapter
}

func (s *StoreTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := startPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("could not start postgres container: %s", err)
	}
	s.container = p

	s.db, err = sql.Open(
		"postgres",
		fmt.Sprintf(
			"postgres://ragserver:ragserver@%s/ragserver?sslmode=disable",
			os.Getenv("POSTGRES_ADDR"),
		),
	)
	s.Require().NoError(err)
}

func (s *StoreTestSuite) TearDownSuite() {
	s.Require().NoError(s.db.Close())
}

func (s *StoreTestSuite) SetupTest() {
	// Migrate down and migrate up to have a clean schema
	driver, err := postgres.WithInstance(s.db, &postgres.Config{SchemaName: "public"})
	s.Require().NoError(err)

	migrationsPath, err := filepath.Abs("../../db/migrations")
	s.Require().NoError(err)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
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

func startPostgresContainer(ctx context.Context) (*dockertest.Resource, error) {
	// Start a new docker pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not construct pool: %w", err)
	}

	// Uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not connect to Docker: %w", err)
	}

	r, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17.6-alpine3.22",
		Env: []string{
			"POSTGRES_DB=ragserver",
			"POSTGRES_USER=ragserver",
			"POSTGRES_PASSWORD=ragserver",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		return nil, fmt.Errorf("could not start resource: %w", err)
	}

	r.Expire(60)

	postgresPort := r.GetPort("5432/tcp")
	addr := fmt.Sprintf("localhost:%s", postgresPort)

	os.Setenv("POSTGRES_ADDR", addr)

	// Wait for the Redis to be ready
	if err := pool.Retry(func() error {
		db, err := sql.Open(
			"postgres",
			fmt.Sprintf(
				"postgres://ragserver:ragserver@%s/ragserver?sslmode=disable",
				addr,
			),
		)
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	return r, nil
}
