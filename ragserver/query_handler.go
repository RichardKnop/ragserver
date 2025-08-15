package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"google.golang.org/genai"
)

const (
	generativeModelName = "gemini-2.5-flash"
	embeddingModelName  = "text-embedding-004"
)

type QueryType string

const (
	QueryTypeText   QueryType = "text"
	QueryTypeMetric QueryType = "metric"
)

type QueryRequest struct {
	Content string    `json:"content"`
	Type    QueryType `json:"type"`
}

type Result struct {
	Responses []Response `json:"responses"`
}

type Response struct {
	Type   QueryType `json:"type"`
	Text   string    `json:"text"`
	Metric float64   `json:"metric,omitempty"` // Only used for QueryTypeMetric
	// Data map[string]any `json:"data,omitempty"`
}

func (rs *ragServer) queryHandler(w http.ResponseWriter, req *http.Request) {
	// Parse HTTP request from JSON.
	qr := new(QueryRequest)
	if err := readRequestJSON(req, qr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("received query: %s", qr.Content)

	switch qr.Type {
	case QueryTypeText, QueryTypeMetric:
	default:
		http.Error(w, "invalid query type", http.StatusBadRequest)
		return
	}

	ctx := req.Context()

	// Embed the query contents.
	embedResponse, err := rs.client.Models.EmbedContent(ctx,
		embeddingModelName,
		[]*genai.Content{genai.NewContentFromText(qr.Content, genai.RoleUser)},
		nil,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Search weaviate to find the most relevant (closest in vector space)
	// documents to the query.
	gql := rs.wvClient.GraphQL()
	graphqlResponse, err := gql.Get().
		WithNearVector(
			gql.NearVectorArgBuilder().WithVector(embedResponse.Embeddings[0].Values)).
		WithClassName("Document").
		WithFields(graphql.Field{Name: "text"}).
		WithLimit(3).
		Do(ctx)
	if werr := combinedWeaviateError(graphqlResponse, err); werr != nil {
		http.Error(w, werr.Error(), http.StatusInternalServerError)
		return
	}

	contents, err := decodeGetResults(graphqlResponse)
	if err != nil {
		http.Error(w, fmt.Errorf("reading weaviate response: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	// Create a RAG query for the LLM with the most relevant documents as
	// context.
	var part string
	switch qr.Type {
	case QueryTypeText:
		part = fmt.Sprintf(ragTemplateStr, qr.Content, strings.Join(contents, "\n"))
	case QueryTypeMetric:
		part = fmt.Sprintf(ragTemplateMetricValue, qr.Content, strings.Join(contents, "\n"))
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
		log.Printf("calling generative model: %v", err.Error())
		http.Error(w, "generative model error", http.StatusInternalServerError)
		return
	}
	if len(resp.Candidates) != 1 {
		log.Printf("got %v candidates, expected 1", len(resp.Candidates))
		http.Error(w, "generative model error", http.StatusInternalServerError)
		return
	}

	log.Printf("gen AI response text: %s", resp.Text())

	var (
		result    Result
		respTexts []string
	)
	if aTest := resp.Text(); strings.TrimSpace(aTest) != "" {
		respTexts = append(respTexts, resp.Text())
	}

	for _, text := range respTexts {
		aResponse := Response{
			Type: qr.Type,
		}

		aText := strings.TrimRight(strings.TrimSpace(text), "\r\n")

		switch qr.Type {
		case QueryTypeText:
			aResponse.Text = aText
		case QueryTypeMetric:
			parts := strings.Split(aText, "\n")
			if len(parts) < 2 {
				http.Error(w, "response not in a valid format for a metric query", http.StatusInternalServerError)
				return
			}

			metricValue, err := strconv.ParseFloat(parts[len(parts)-1], 64)
			if err != nil {
				log.Printf("could not parse %s as float", aResponse.Text)
				http.Error(w, "could not parse metric value", http.StatusInternalServerError)
				return
			}
			aResponse.Metric = metricValue

			aResponse.Text = strings.Join(parts[:len(parts)-1], "\n")
		}

		result.Responses = append(result.Responses, aResponse)
	}

	log.Printf("result: %+v", result)
	renderJSON(w, result)
}

// decodeGetResults decodes the result returned by Weaviate's GraphQL Get
// query; these are returned as a nested map[string]any (just like JSON
// unmarshaled into a map[string]any). We have to extract all document contents
// as a list of strings.
func decodeGetResults(graphqlResponse *models.GraphQLResponse) ([]string, error) {
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
