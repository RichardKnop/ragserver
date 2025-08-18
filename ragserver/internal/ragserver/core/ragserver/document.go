package ragserver

import (
	"context"
	"fmt"
	"log"

	"github.com/RichardKnop/ai/ragserver/internal/pkg/authz"
)

type Document struct {
	Text   string `json:"text"`
	FileID FileID `json:"-"`
}

func (rs *ragServer) CreateDocuments(ctx context.Context, principal authz.Principal, documents []Document) error {
	// Use the batch embedding API to embed all documents at once.
	vectors, err := rs.genai.EmbedDocuments(ctx, documents)
	if err != nil {
		return fmt.Errorf("error generating vectors: %v", err)
	}

	log.Printf("generated vectors: %d", len(vectors))

	if err := rs.weaviate.SaveEmbeddings(ctx, documents, vectors); err != nil {
		return fmt.Errorf("error saving embeddings: %v", err)
	}

	return nil
}
