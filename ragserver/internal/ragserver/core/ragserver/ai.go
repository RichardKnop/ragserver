package ragserver

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/RichardKnop/ai/ragserver/internal/pkg/authz"
)

type Vector []float32

type QueryType string

const (
	QueryTypeText   QueryType = "text"
	QueryTypeMetric QueryType = "metric"
)

type Query struct {
	Type QueryType `json:"type"`
	Text string    `json:"text"`
}

type Response struct {
	Type   QueryType `json:"type"`
	Text   string    `json:"text"`
	Metric float64   `json:"metric,omitempty"` // Only used for QueryTypeMetric
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
	contents, err := rs.weaviate.SearchDocuments(ctx, vector)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %v", err)
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("no documents found for query: %s", query)
	}

	respTexts, err := rs.generateResponses(ctx, query, contents)
	if err != nil {
		return nil, fmt.Errorf("error generating responses: %v", err)
	}

	log.Printf("gen AI response texts: %s", respTexts)

	var queryResponses []Response

	for _, text := range respTexts {
		aResponse := Response{
			Type: query.Type,
		}

		aText := strings.TrimRight(strings.TrimSpace(text), "\r\n")

		switch query.Type {
		case QueryTypeText:
			aResponse.Text = aText
		case QueryTypeMetric:
			parts := strings.Split(aText, "\n")
			if len(parts) < 2 {
				return nil, fmt.Errorf("response not in a valid format for a metric query")
			}

			metricValue, err := strconv.ParseFloat(parts[len(parts)-1], 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse metric value: %v", err)
			}
			aResponse.Metric = metricValue

			aResponse.Text = strings.Join(parts[:len(parts)-1], "\n")
		}

		queryResponses = append(queryResponses, aResponse)
	}

	log.Printf("result: %+v", queryResponses)
	return queryResponses, nil
}

func (rs *ragServer) generateResponses(ctx context.Context, query Query, contexts []string) ([]string, error) {
	// Create a RAG query for the LLM with the most relevant documents as context.
	var input string
	switch query.Type {
	case QueryTypeText:
		input = fmt.Sprintf(ragTemplateStr, query.Text, strings.Join(contexts, "\n"))
	case QueryTypeMetric:
		input = fmt.Sprintf(ragTemplateMetricValue, query.Text, strings.Join(contexts, "\n"))
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	log.Printf("gen AI query input: %s", input)

	responses, err := rs.genai.Generate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}

	return responses, nil
}
