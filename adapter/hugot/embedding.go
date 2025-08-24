package hugot

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) EmbedDocuments(ctx context.Context, documents []ragserver.Document) ([]ragserver.Vector, error) {
	sentences := make([]string, 0, len(documents))
	for _, aDocument := range documents {
		sentences = append(sentences, aDocument.Content)
	}

	embeddingResult, err := a.pipeline.RunPipeline(sentences)
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
	embeddingResult, err := a.pipeline.RunPipeline([]string{content})
	if err != nil {
		return ragserver.Vector{}, err
	}
	return embeddingResult.Embeddings[0], nil
}

func floatsToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)

	for i, f := range fs {
		u := math.Float32bits(f)
		binary.NativeEndian.PutUint32(buf[i*4:], u)
	}

	return buf
}
