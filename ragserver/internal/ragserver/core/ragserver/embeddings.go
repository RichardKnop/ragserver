package ragserver

import (
	"context"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"google.golang.org/genai"
)

type Vector []float32

func (rs *ragServer) embedDocuments(ctx context.Context, documents []Document) ([]Vector, error) {
	// Use the batch embedding API to embed all documents at once.
	contents := make([]*genai.Content, 0, len(documents))
	for _, aDocument := range documents {
		contents = append(contents, genai.NewContentFromText(aDocument.Text, genai.RoleUser))
	}
	embedResponse, err := rs.client.Models.EmbedContent(ctx,
		embeddingModelName,
		contents,
		nil,
	)
	log.Printf("invoking embedding model with %v documents", len(documents))
	if err != nil {
		return nil, fmt.Errorf("embed content error: %w", err)
	}

	if len(embedResponse.Embeddings) != len(documents) {
		return nil, fmt.Errorf("embedded batch size mismatch")
	}

	vectors := make([]Vector, 0, len(embedResponse.Embeddings))

	for i := range embedResponse.Embeddings {
		vectors = append(vectors, embedResponse.Embeddings[i].Values)
	}

	return vectors, nil
}

func (rs *ragServer) saveEmbeddings(ctx context.Context, documents []Document, vectors []Vector) error {
	// Convert our documents - along with their embedding vectors - into types
	// used by the Weaviate client library.
	objects := make([]*models.Object, len(documents))
	for i, doc := range documents {
		if len(vectors[i]) == 0 {
			return fmt.Errorf("empty vector")
		}
		objects[i] = &models.Object{
			Class: "Document",
			Properties: map[string]any{
				"text": doc.Text,
			},
			Vector: models.C11yVector(vectors[i]),
		}
	}

	// Store documents with embeddings in the Weaviate DB.
	_, err := rs.wvClient.Batch().ObjectsBatcher().WithObjects(objects...).Do(ctx)
	if err != nil {
		return err
	}

	log.Printf("stored %v objects in weaviate", len(objects))
	return err
}

func (rs *ragServer) embedContent(ctx context.Context, content string) (Vector, error) {
	embedResponse, err := rs.client.Models.EmbedContent(ctx,
		embeddingModelName,
		[]*genai.Content{genai.NewContentFromText(content, genai.RoleUser)},
		nil,
	)
	if err != nil {
		return Vector{}, err
	}
	return embedResponse.Embeddings[0].Values, nil
}

func (rs *ragServer) searchDocuments(ctx context.Context, vector Vector) ([]string, error) {
	gql := rs.wvClient.GraphQL()
	nearVector := gql.NearVectorArgBuilder().WithVector([]float32(vector))
	graphqlResponse, err := gql.Get().
		WithNearVector(nearVector).
		WithClassName("Document").
		WithFields(graphql.Field{Name: "text"}).
		WithLimit(3).
		Do(ctx)
	if err := combinedWeaviateError(graphqlResponse, err); err != nil {
		return nil, err
	}

	return decodeGetDocumentResults(graphqlResponse)
}

// decodeGetResults decodes the result returned by Weaviate's GraphQL Get
// query; these are returned as a nested map[string]any (just like JSON
// unmarshaled into a map[string]any). We have to extract all document contents
// as a list of strings.
func decodeGetDocumentResults(graphqlResponse *models.GraphQLResponse) ([]string, error) {
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

	var out []string
	for _, s := range slc {
		smap, ok := s.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid element in list of documents")
		}
		s, ok := smap["text"].(string)
		if !ok {
			return nil, fmt.Errorf("expected string in list of documents")
		}
		out = append(out, s)
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
