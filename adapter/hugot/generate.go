package hugot

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/knights-analytics/hugot/pipelines"

	"github.com/RichardKnop/ragserver"
)

type MetricValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Response struct {
	Text              string      `json:"text"`
	Metric            MetricValue `json:"metric"`
	Boolean           bool        `json:"boolean"`
	RelevantDocuments []string    `json:"relevant_documents"`
}

func (a *Adapter) Generate(ctx context.Context, query ragserver.Query, documents []ragserver.Document) ([]ragserver.Response, error) {
	contexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		contexts = append(contexts, strconv.Quote(strings.TrimSpace(doc.Content)))
	}

	// Create a RAG query for the LLM with the most relevant documents as context.
	var prompt string
	switch query.Type {
	case ragserver.QueryTypeText:
		prompt = fmt.Sprintf(ragTemplateStr, query.Text, strings.Join(contexts, "\n"))
	case ragserver.QueryTypeMetric:
		prompt = fmt.Sprintf(ragTemplateMetricValue, query.Text, strings.Join(contexts, "\n"))
	case ragserver.QueryTypeBoolean:
		prompt = fmt.Sprintf(ragTemplateBooleanValue, query.Text, strings.Join(contexts, "\n"))
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	log.Println("genai prompt:", prompt)

	batchResult, err := a.generative.RunWithTemplate([][]pipelines.Message{
		{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}
	if len(batchResult.GetOutput()) != 1 {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}

	result := batchResult.GetOutput()[0].(string)

	log.Println("genai response:", result)

	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimSuffix(result, "```")

	structuredResp := Response{}
	if err := json.Unmarshal([]byte(result), &structuredResp); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %v", err)
	}

	response := ragserver.Response{
		Text: structuredResp.Text,
	}

	switch query.Type {
	case ragserver.QueryTypeMetric:
		response.Metric = ragserver.MetricValue{
			Value: structuredResp.Metric.Value,
			Unit:  structuredResp.Metric.Unit,
		}
	case ragserver.QueryTypeBoolean:
		response.Boolean = ragserver.BooleanValue(structuredResp.Boolean)
	}

	documentMap := make(map[string]ragserver.Document)
	for _, doc := range documents {
		hash := md5.Sum([]byte(strings.TrimSpace(doc.Content)))
		documentMap[string(hash[:])] = doc
	}

	for _, docTxt := range structuredResp.RelevantDocuments {
		hash := md5.Sum([]byte(strings.TrimSpace(docTxt)))
		doc, ok := documentMap[string(hash[:])]
		if !ok {
			log.Printf("could not find document for: %s", docTxt)
			continue
		}
		response.Documents = append(response.Documents, doc)
	}

	return []ragserver.Response{response}, nil
}
