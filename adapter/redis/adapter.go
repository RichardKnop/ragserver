package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

type Adapter struct {
	client               *redis.Client
	indexName            string
	indexPrefix          string
	dialectVersion       int
	vectorDim            int
	vectorDistanceMetric string
}

type Option func(*Adapter)

const (
	defaultIndexName            = "vector-idx"
	defaultIndexPrefix          = "doc:"
	defaultDialectVersion       = 2
	defaultVectorDim            = 768
	defaultVectorDistanceMetric = "L2"
)

func New(ctx context.Context, client *redis.Client, options ...Option) (*Adapter, error) {
	a := &Adapter{
		client:               client,
		indexPrefix:          defaultIndexPrefix,
		indexName:            defaultIndexName,
		dialectVersion:       defaultDialectVersion,
		vectorDim:            defaultVectorDim,
		vectorDistanceMetric: defaultVectorDistanceMetric,
	}

	for _, o := range options {
		o(a)
	}

	// Append vector dim to index name to allow multiple indexes with different dimensions
	// e.g. text-embedding-004 produces 768-dimensional vectors by default
	// but allows a developer to choose any number of dimensions between 1 and 768
	// However, all-MiniLM-L6-v2 maps sentences & paragraphs to a 384 dimensional dense vector space
	// so we might want to create a separate index for that
	a.indexName = fmt.Sprintf("%s_dim%d", a.indexName, a.vectorDim)

	log.Println(
		"init redis adapter,",
		"index name:", a.indexName,
		"prefix:", a.indexPrefix,
		"dialect version:", a.dialectVersion,
		"vector dim:", a.vectorDim,
		"vector distance metric:", a.vectorDistanceMetric,
	)

	return a, a.init(ctx)
}

func WithIndexName(indexName string) Option {
	return func(a *Adapter) {
		a.indexName = indexName
	}
}

func WithIndexPrefix(prefix string) Option {
	return func(a *Adapter) {
		a.indexPrefix = prefix
	}
}

func WithDialectVersion(version int) Option {
	return func(a *Adapter) {
		a.dialectVersion = version
	}
}

func WithVectorDim(dim int) Option {
	return func(a *Adapter) {
		a.vectorDim = dim
	}
}

func WithVectorDistanceMetric(metric string) Option {
	return func(a *Adapter) {
		a.vectorDistanceMetric = metric
	}
}

const adapterName = "redis"

func (a *Adapter) Name() string {
	return adapterName
}

func (a *Adapter) init(ctx context.Context) error {
	// if err := a.dropIndex(ctx); err != nil {
	// 	return err
	// }
	// return nil
	indexes, err := a.client.FT_List(ctx).Result()
	if err != nil {
		return err
	}
	for _, existingIndex := range indexes {
		if existingIndex == a.indexName {
			log.Println("redis index already exists:", a.indexName)
			return nil
		}
	}
	return a.createIndex(ctx)
}

func (a *Adapter) dropIndex(ctx context.Context) error {
	_, err := a.client.FTDropIndexWithArgs(ctx,
		a.indexName,
		&redis.FTDropIndexOptions{
			DeleteDocs: true,
		},
	).Result()
	if err != nil {
		return err
	}
	log.Println("dropped redis index:", a.indexName)
	return nil
}

func (a *Adapter) createIndex(ctx context.Context) error {
	_, err := a.client.FTCreate(ctx,
		a.indexName,
		&redis.FTCreateOptions{
			OnHash: true,
			Prefix: []any{a.indexPrefix},
		},
		&redis.FieldSchema{
			FieldName: "content",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "file_id",
			FieldType: redis.SearchFieldTypeTag,
		},
		&redis.FieldSchema{
			FieldName: "page",
			FieldType: redis.SearchFieldTypeTag,
		},
		&redis.FieldSchema{
			FieldName: "embedding",
			FieldType: redis.SearchFieldTypeVector,
			VectorArgs: &redis.FTVectorArgs{
				HNSWOptions: &redis.FTHNSWOptions{
					Dim:            a.vectorDim,
					DistanceMetric: a.vectorDistanceMetric, // "COSINE", "IP", "L2"
					Type:           "FLOAT32",
				},
			},
		},
	).Result()
	if err != nil {
		return fmt.Errorf("error creating redis index: %v", err)
	}
	log.Println("created redis index:", a.indexName)
	return nil
}
