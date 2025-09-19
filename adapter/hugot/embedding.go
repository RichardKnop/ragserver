package hugot

import (
	"context"
	"fmt"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) EmbedDocuments(ctx context.Context, documents []ragserver.Document) ([]ragserver.Vector, error) {
	if a.embedding == nil {
		return nil, fmt.Errorf("embedding pipeline not initialized")
	}

	sentences := make([]string, 0, len(documents))
	for _, aDocument := range documents {
		sentences = append(sentences, aDocument.Content)
	}

	embeddingResult, err := a.embedding.RunPipeline(sentences)
	if err != nil {
		return nil, err
	}

	embeddings := embeddingResult.Embeddings

	if len(embeddings) != len(documents) {
		return nil, fmt.Errorf("embedded batch size mismatch")
	}

	vectors := make([]ragserver.Vector, 0, len(embeddings))

	for i := range embeddings {
		vectors = append(vectors, embeddings[i])
	}

	return vectors, nil
}

func (a *Adapter) EmbedContent(ctx context.Context, content string) (ragserver.Vector, error) {
	if a.embedding == nil {
		return nil, fmt.Errorf("embedding pipeline not initialized")
	}

	embeddingResult, err := a.embedding.RunPipeline([]string{content})
	if err != nil {
		return ragserver.Vector{}, err
	}
	return embeddingResult.Embeddings[0], nil
}
