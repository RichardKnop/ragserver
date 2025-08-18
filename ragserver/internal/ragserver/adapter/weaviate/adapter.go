package weaviate

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
)

type Adapter struct {
	client *weaviate.Client
}

type Option func(*Adapter)

func New(ctx context.Context, client *weaviate.Client, options ...Option) (*Adapter, error) {
	a := &Adapter{
		client: client,
	}

	for _, o := range options {
		o(a)
	}

	return a, a.init(ctx)
}

const className = "Document"

func (a *Adapter) init(ctx context.Context) error {
	// Create a new class (collection) in weaviate if it doesn't exist yet.
	cls := &models.Class{
		Class:      className,
		Vectorizer: "none",
	}
	exists, err := a.client.Schema().ClassExistenceChecker().WithClassName(cls.Class).Do(ctx)
	if err != nil {
		return fmt.Errorf("weaviate error: %w", err)
	}
	if !exists {
		err = a.client.Schema().ClassCreator().WithClass(cls).Do(ctx)
		if err != nil {
			return fmt.Errorf("weaviate error: %w", err)
		}
	}

	return nil
}
