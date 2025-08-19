package ragserver

import (
	"context"
	"fmt"
	"log"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
)

type Vector []float32

type QueryType string

const (
	QueryTypeText   QueryType = "text"
	QueryTypeMetric QueryType = "metric"
)

type Query struct {
	Type QueryType
	Text string
}

type MetricValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Response struct {
	Text      string      `json:"text"`
	Metric    MetricValue `json:"metric"`
	Documents []Document  `json:"-"`
}

func (rs *ragServer) Generate(ctx context.Context, principal authz.Principal, query Query, fileIDs ...FileID) ([]Response, error) {
	switch query.Type {
	case QueryTypeText, QueryTypeMetric:
	default:
		return nil, fmt.Errorf("invalid query type: %s", query.Type)
	}

	log.Printf("received query: %s, file IDs: %v", query, fileIDs)

	// Check all file IDs exist in the database
	for _, fileID := range fileIDs {
		_, err := rs.store.FindFile(ctx, fileID)
		if err != nil {
			return nil, fmt.Errorf("error finding file: %v", err)
		}
	}

	// Embed the query contents.
	vector, err := rs.genai.EmbedContent(ctx, query.Text)
	if err != nil {
		return nil, fmt.Errorf("embedding query content: %v", err)
	}

	// Search weaviate to find the most relevant (closest in vector space)
	// documents to the query.
	documents, err := rs.weaviate.SearchDocuments(ctx, vector, fileIDs...)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %v", err)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents found for query: %s", query)
	}

	responses, err := rs.genai.Generate(ctx, query, documents)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}

	return responses, nil
}
