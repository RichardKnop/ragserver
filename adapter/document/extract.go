package document

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/neurosnap/sentences"
	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver"
)

const summarizePrompt = `
Summarize each page of this document. For each page, provide a full summary including 
all the data from text on the page and all the data from tables on the page.
Response is a JSON array, with each item being a full summary of a pge.
`

func (a *Adapter) Extract(ctx context.Context, tempFile io.ReadSeeker, topics ragserver.RelevantTopics) ([]ragserver.Document, error) {
	documentBytes, err := io.ReadAll(tempFile)
	if err != nil {
		return nil, err
	}

	parts := []*genai.Part{
		{
			InlineData: &genai.Blob{
				MIMEType: "application/pdf",
				Data:     documentBytes,
			},
		},
		genai.NewPartFromText(summarizePrompt),
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: nil, // Disables thinking
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
		},
	}

	result, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		contents,
		config,
	)
	if err != nil {
		return nil, err
	}

	response := []string{}
	if err := json.Unmarshal([]byte(result.Text()), &response); err != nil {
		return nil, err
	}

	var (
		// Create the default sentence tokenizer
		tokenizer  = sentences.NewSentenceTokenizer(a.training)
		documents  = make([]ragserver.Document, 0, 100)
		numPages   = len(response)
		topicCount = map[string]int{}
	)

	for i, page := range response {
		pageNum := i + 1
		log.Printf("processing page %d/%d", pageNum, numPages)

		for _, aSentence := range tokenizer.Tokenize(page) {
			if len(topics) > 0 {
				aTopic, ok := topics.IsRelevant(aSentence.Text)
				if !ok {
					continue
				}
				if aTopic.Name != "" {
					_, ok := topicCount[aTopic.Name]
					if !ok {
						topicCount[aTopic.Name] = 0
					}
					topicCount[aTopic.Name] += 1
				}
			}

			documents = append(documents, ragserver.Document{
				Content: strings.TrimSpace(aSentence.Text),
				Page:    i + 1,
			})
		}
	}

	for name, count := range topicCount {
		log.Printf("%s relevant sentences: %d", name, count)
	}

	log.Printf("number of documents: %d", len(documents))

	return documents, nil
}
