package weaviate

import (
	"context"
	"fmt"
	"log"

	"github.com/gofrs/uuid/v5"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

func (a *Adapter) SaveEmbeddings(ctx context.Context, documents []ragserver.Document, vectors []ragserver.Vector) error {
	// Convert our documents - along with their embedding vectors - into types
	// used by the Weaviate client library.
	objects := make([]*models.Object, len(documents))
	for i, doc := range documents {
		if len(vectors[i]) == 0 {
			return fmt.Errorf("empty vector")
		}
		properties := map[string]any{
			"text": doc.Text,
			"page": doc.Page,
		}
		if !doc.FileID.IsNil() {
			properties["file_id"] = doc.FileID.String()
		}
		objects[i] = &models.Object{
			Class:      className,
			Properties: properties,
			Vector:     models.C11yVector(vectors[i]),
		}
	}

	// Store documents with embeddings in the Weaviate DB.
	_, err := a.client.Batch().ObjectsBatcher().WithObjects(objects...).Do(ctx)
	if err != nil {
		return err
	}

	log.Printf("stored %v objects in weaviate", len(objects))
	return err
}

func (a *Adapter) SearchDocuments(ctx context.Context, vector ragserver.Vector, fileIDs ...ragserver.FileID) ([]ragserver.Document, error) {
	gql := a.client.GraphQL()
	nearVector := gql.NearVectorArgBuilder().WithVector([]float32(vector))

	builder := gql.Get().
		WithNearVector(nearVector).
		WithClassName("Document").
		WithFields(
			graphql.Field{Name: "text"},
			graphql.Field{Name: "page"},
			graphql.Field{Name: "file_id"},
		).
		WithLimit(10)

	if len(fileIDs) > 0 {
		filter := filters.Where()
		filter.WithOperator(filters.ContainsAny)
		filter.WithPath([]string{"file_id"})
		filter.WithValueString(fileIDsToStrings(fileIDs)...)
		builder = builder.WithWhere(filter)
	}

	graphqlResponse, err := builder.Do(ctx)
	if err := combinedWeaviateError(graphqlResponse, err); err != nil {
		return nil, err
	}

	return decodeGetDocumentResults(graphqlResponse)
}

func fileIDsToStrings(fileIDs []ragserver.FileID) []string {
	ids := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		ids = append(ids, fileID.String())
	}
	return ids
}

// decodeGetResults decodes the result returned by Weaviate's GraphQL Get
// query; these are returned as a nested map[string]any (just like JSON
// unmarshaled into a map[string]any). We have to extract all document contents
// as a list of strings.
func decodeGetDocumentResults(graphqlResponse *models.GraphQLResponse) ([]ragserver.Document, error) {
	data, ok := graphqlResponse.Data["Get"]
	if !ok {
		return nil, fmt.Errorf("get key not found in result")
	}
	doc, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("get key unexpected type")
	}
	slc, ok := doc["Document"].([]any)
	if !ok {
		return nil, fmt.Errorf("document is not a list of results")
	}

	var out []ragserver.Document
	for _, s := range slc {
		smap, ok := s.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid element in list of documents")
		}
		text, ok := smap["text"].(string)
		if !ok {
			return nil, fmt.Errorf("expected text in document")
		}
		page, ok := smap["page"].(float64)
		if !ok {
			return nil, fmt.Errorf("expected page in document")
		}
		id, ok := smap["file_id"].(string)
		if !ok {
			return nil, fmt.Errorf("expected file_id in document")
		}
		fileID, err := uuid.FromString(id)
		if err != nil {
			return nil, fmt.Errorf("invalid file_id in document: %w", err)
		}
		out = append(out, ragserver.Document{
			Text:   text,
			Page:   int(page),
			FileID: ragserver.FileID{UUID: fileID},
		})
	}
	return out, nil
}

// combinedWeaviateError generates an error if err is non-nil or result has
// errors, and returns an error (or nil if there's no error). It's useful for
// the results of the Weaviate GraphQL API's "Do" calls.
func combinedWeaviateError(graphqlResponse *models.GraphQLResponse, err error) error {
	if err != nil {
		return err
	}
	if len(graphqlResponse.Errors) != 0 {
		var ss []string
		for _, e := range graphqlResponse.Errors {
			ss = append(ss, e.Message)
		}
		return fmt.Errorf("weaviate error: %v", ss)
	}
	return nil
}
