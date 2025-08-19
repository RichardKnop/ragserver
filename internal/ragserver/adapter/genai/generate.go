package genai

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

var (
	textSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"text": {Type: genai.TypeString},
			"relevant_documents": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeInteger,
				},
			},
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
			"relevant_documents": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
		},
	}
)

type Response struct {
	ragserver.Response
	RelevantDocuments []string `json:"relevant_documents"`
}

func (a *Adapter) Generate(ctx context.Context, query ragserver.Query, documents []ragserver.Document) ([]ragserver.Response, error) {
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: nil, // Disables thinking
		},
	}

	contexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		contexts = append(contexts, strconv.Quote(doc.Text))
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

	response := Response{}
	if err := json.Unmarshal([]byte(resp.Text()), &response); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %v", err)
	}

	documentMap := make(map[string]ragserver.Document)
	for _, doc := range documents {
		hash := md5.Sum([]byte(doc.Text))
		documentMap[string(hash[:])] = doc
	}

	for _, docTxt := range response.RelevantDocuments {
		hash := md5.Sum([]byte(docTxt))
		doc, ok := documentMap[string(hash[:])]
		if !ok {
			log.Printf("could not find document for: %s", docTxt)
			continue
		}
		response.Documents = append(response.Documents, doc)
	}

	return []ragserver.Response{response.Response}, nil
}
