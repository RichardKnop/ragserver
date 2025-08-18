package genai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

func (a *Adapter) EmbedDocuments(ctx context.Context, documents []ragserver.Document) ([]ragserver.Vector, error) {
	// Use the batch embedding API to embed all documents at once.
	contents := make([]*genai.Content, 0, len(documents))
	for _, aDocument := range documents {
		contents = append(contents, genai.NewContentFromText(aDocument.Text, genai.RoleUser))
	}
	embedResponse, err := a.client.Models.EmbedContent(ctx,
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

	vectors := make([]ragserver.Vector, 0, len(embedResponse.Embeddings))

	for i := range embedResponse.Embeddings {
		vectors = append(vectors, embedResponse.Embeddings[i].Values)
	}

	return vectors, nil
}

func (a *Adapter) EmbedContent(ctx context.Context, content string) (ragserver.Vector, error) {
	embedResponse, err := a.client.Models.EmbedContent(ctx,
		embeddingModelName,
		[]*genai.Content{genai.NewContentFromText(content, genai.RoleUser)},
		nil,
	)
	if err != nil {
		return ragserver.Vector{}, err
	}
	return embedResponse.Embeddings[0].Values, nil
}

func (a *Adapter) Generate(ctx context.Context, input string) ([]string, error) {
	resp, err := a.client.Models.GenerateContent(
		ctx,
		a.generativeModelName,
		genai.Text(input),
		&genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingBudget: nil, // Disables thinking
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}
	if len(resp.Candidates) != 1 {
		return nil, fmt.Errorf("got %v candidates, expected 1", len(resp.Candidates))
	}

	var respTexts []string
	if aTest := resp.Text(); strings.TrimSpace(aTest) != "" {
		respTexts = append(respTexts, resp.Text())
	}

	return respTexts, nil
}
