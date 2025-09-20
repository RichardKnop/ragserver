package hugot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/knights-analytics/hugot/pipelines"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) Generate(ctx context.Context, question ragserver.Question, documents []ragserver.Document) ([]ragserver.Response, error) {
	contexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		contexts = append(contexts, doc.Content)
	}

	var template string
	switch question.Type {
	case ragserver.QuestionTypeText:
		templateBytes, err := os.ReadFile(path.Join(a.templatesDir, "text.tmpl"))
		if err != nil {
			return nil, fmt.Errorf("reading text template: %w", err)
		}
		template = string(templateBytes)
	case ragserver.QuestionTypeMetric:
		templateBytes, err := os.ReadFile(path.Join(a.templatesDir, "metric.tmpl"))
		if err != nil {
			return nil, fmt.Errorf("reading metric template: %w", err)
		}
		template = string(templateBytes)
	case ragserver.QuestionTypeBoolean:
		templateBytes, err := os.ReadFile(path.Join(a.templatesDir, "boolean.tmpl"))
		if err != nil {
			return nil, fmt.Errorf("reading boolean template: %w", err)
		}
		template = string(templateBytes)
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	// Create a RAG query for the LLM with the most relevant documents as context.
	prompt := fmt.Sprintf(template, question.Content, strings.Join(contexts, "\n"))

	a.logger.Sugar().With("question", question.Content).Info("generating answer")

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

	a.logger.Sugar().Infof("genai response: %s", result)

	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimSuffix(result, "```")

	modelResponse := ragserver.ModelResponse{}
	if err := json.Unmarshal([]byte(result), &modelResponse); err != nil {
		return nil, fmt.Errorf("unmarshalling model response: %v", err)
	}

	response := ragserver.Response{
		Text: modelResponse.Text,
	}

	switch question.Type {
	case ragserver.QuestionTypeMetric:
		response.Metric = modelResponse.Metric
	case ragserver.QuestionTypeBoolean:
		response.Boolean = modelResponse.Boolean
	}

	matchedDocuments, unmatchedSnippets := ragserver.MatchSnippetsToDocuments(modelResponse.RelevantSnippets, documents)
	if len(unmatchedSnippets) > 0 {
		a.logger.Sugar().Warnf("unmatched snippets: %v", unmatchedSnippets)
	}
	response.Documents = matchedDocuments

	return []ragserver.Response{response}, nil
}
