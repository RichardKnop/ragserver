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
	// _, derr := a.client.FTDropIndexWithArgs(ctx,
	// 	a.indexName,
	// 	&redis.FTDropIndexOptions{
	// 		DeleteDocs: true,
	// 	},
	// ).Result()
	// if derr != nil {
	// 	return derr
	// }
	// log.Println("dropped redis index:", a.indexName)
	// return nil
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
					// Google text-embedding-004 produces vectors with 768 dimensions by default,
					// but allows a developer to choose any number of dimensions between 1 and 768
					// However, all-MiniLM-L6-v2 maps sentences & paragraphs to a 384 dimensional
					// dense vector space
					Dim:            a.vectorDim,
					DistanceMetric: a.vectorDistanceMetric, // "COSINE", "IP", "L2"
					Type:           "FLOAT32",
				},
			},
		},
	).Result()
	if err != nil {
		if err.Error() == "Index already exists" {
			log.Println("redis index already exists:", a.indexName)
			return nil
		}
		return fmt.Errorf("error creating redis index: %v", err)
	}
	log.Println("created redis index:", a.indexName)
	return nil
}
