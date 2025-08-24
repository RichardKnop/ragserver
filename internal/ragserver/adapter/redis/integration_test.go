package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}

type RedisTestSuite struct {
	suite.Suite
	container *dockertest.Resource
	client    *redis.Client
	adapter   *Adapter
}

func (s *RedisTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := startRedisContainer(ctx)
	if err != nil {
		log.Fatalf("could not start redis container: %s", err)
	}
	s.container = r

	s.client = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		DB:       0,
		Protocol: 2,
	})
}

func (s *RedisTestSuite) TearDownSuite() {
	err := s.container.Close()
	s.Require().NoError(err)
}

func (s *RedisTestSuite) SetupTest() {
	ctx, cancel := testContext()
	defer cancel()

	err := s.client.FlushDB(ctx).Err()
	s.Require().NoError(err)

	s.adapter, err = New(
		ctx,
		s.client,
		WithIndexName("text-idx"),
		WithIndexPrefix("doc:"),
		WithDialectVersion(2),
		WithVectorDim(768),
		WithVectorDistanceMetric("L2"),
	)
	s.Require().NoError(err)
}

func (s *RedisTestSuite) TearDownTest() {
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

func startRedisContainer(ctx context.Context) (*dockertest.Resource, error) {
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
		Repository: "redis/redis-stack-server",
		Tag:        "7.2.0-v18",
		Env:        []string{},
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

	r.Expire(10)

	redisPort := r.GetPort("6379/tcp")
	addr := fmt.Sprintf("localhost:%s", redisPort)

	os.Setenv("REDIS_ADDR", addr)

	// Wait for the Redis to be ready
	if err := pool.Retry(func() error {
		result, err := redis.NewClient(&redis.Options{
			Addr:     addr,
			DB:       0,
			Protocol: 2,
		}).Ping(ctx).Result()
		if err != nil {
			return err
		}
		if result != "PONG" {
			return fmt.Errorf("unexpected redis ping response: %s", result)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %w", err)
	}

	return r, nil
}
