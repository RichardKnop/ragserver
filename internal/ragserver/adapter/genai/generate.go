package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

var (
	textSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"text": {Type: genai.TypeString},
		},
	}

	metricSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"text": {Type: genai.TypeString},
			"metric": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"value": {Type: genai.TypeNumber},
					"unit":  {Type: genai.TypeString},
				},
			},
		},
	}
)

func (a *Adapter) Generate(ctx context.Context, query ragserver.Query, documents []ragserver.Document) ([]ragserver.Response, error) {
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: nil, // Disables thinking
		},
	}

	contexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		contexts = append(contexts, doc.Text)
	}

	switch query.Type {
	case ragserver.QueryTypeText:
		config.ResponseSchema = textSchema
	case ragserver.QueryTypeMetric:
		config.ResponseSchema = metricSchema
	}

	// Create a RAG query for the LLM with the most relevant documents as context.
	var prompt string
	switch query.Type {
	case ragserver.QueryTypeText:
		prompt = fmt.Sprintf(ragTemplateStr, query.Text, strings.Join(contexts, "\n"))
	case ragserver.QueryTypeMetric:
		prompt = fmt.Sprintf(ragTemplateMetricValue, query.Text, strings.Join(contexts, "\n"))
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	log.Println("genai prompt:", prompt)

	resp, err := a.client.Models.GenerateContent(
		ctx,
		a.generativeModelName,
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}
	if len(resp.Candidates) != 1 {
		return nil, fmt.Errorf("got %v candidates, expected 1", len(resp.Candidates))
	}

	log.Println("genai response:", resp.Text())

	response := ragserver.Response{}
	if err := json.Unmarshal([]byte(resp.Text()), &response); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %v", err)
	}

	return []ragserver.Response{response}, nil
}
