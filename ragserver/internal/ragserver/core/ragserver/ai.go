package ragserver

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"google.golang.org/genai"
)

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

func (rs *ragServer) Generate(ctx context.Context, query Query) ([]Response, error) {
	switch query.Type {
	case QueryTypeText, QueryTypeMetric:
	default:
		return nil, fmt.Errorf("invalid query type: %s", query.Type)
	}

	log.Printf("received query: %s", query)

	// Embed the query contents.
	vector, err := rs.embedContent(ctx, query.Text)
	if err != nil {
		return nil, fmt.Errorf("embedding query content: %v", err)
	}

	// Search weaviate to find the most relevant (closest in vector space)
	// documents to the query.
	contents, err := rs.searchDocuments(ctx, vector)
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
	// Create a RAG query for the LLM with the most relevant documents as
	// context.
	var part string
	switch query.Type {
	case QueryTypeText:
		part = fmt.Sprintf(ragTemplateStr, query, strings.Join(contexts, "\n"))
	case QueryTypeMetric:
		part = fmt.Sprintf(ragTemplateMetricValue, query, strings.Join(contexts, "\n"))
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	log.Printf("gen AI query part: %s", part)

	resp, err := rs.client.Models.GenerateContent(
		ctx,
		generativeModelName,
		genai.Text(part),
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
