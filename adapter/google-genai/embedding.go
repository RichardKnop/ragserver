package googlegenai

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) EmbedDocuments(ctx context.Context, documents []ragserver.Document) ([]ragserver.Vector, error) {
	// Use the batch embedding API to embed all documents at once.
	contents := make([]*genai.Content, 0, len(documents))
	for _, aDocument := range documents {
		contents = append(contents, genai.NewContentFromText(aDocument.Content, genai.RoleUser))
	}
	embedResponse, err := a.client.Models.EmbedContent(ctx,
		a.embeddingModel,
		contents,
		nil,
	)
	a.logger.Sugar().Infof("invoking embedding model with %d documents", len(documents))
	if err != nil {
		return nil, fmt.Errorf("embed content error: %w", err)
	}

	if len(embedResponse.Embeddings) != len(documents) {
		return nil, fmt.Errorf("embedded batch size mismatch")
	}

	vectors := make([]ragserver.Vector, 0, len(embedResponse.Embeddings))

	for i := range embedResponse.Embeddings {
		vectors = append(vectors, embedResponse.Embeddings[i].Values)
	}

	return vectors, nil
}

func (a *Adapter) EmbedContent(ctx context.Context, content string) (ragserver.Vector, error) {
	embedResponse, err := a.client.Models.EmbedContent(ctx,
		a.embeddingModel,
		[]*genai.Content{genai.NewContentFromText(content, genai.RoleUser)},
		nil,
	)
	if err != nil {
		return ragserver.Vector{}, err
	}
	return embedResponse.Embeddings[0].Values, nil
}
