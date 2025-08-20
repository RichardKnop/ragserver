package document

import (
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
	"github.com/neurosnap/sentences"
	"google.golang.org/genai"
)

const summarizePrompt = `
Summarize each page of this document. For each page, provide a full summary including 
all the data from text on the page and all the data from tables on the page.
Response is a JSON array, with each item being a full summary of a pge.
`

func (a *Adapter) Extract(ctx context.Context, tempFile io.ReadSeeker) ([]ragserver.Document, error) {
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
		a.generativeModelName,
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
		numPages  = len(response)
		tokenizer = sentences.NewSentenceTokenizer(a.training)
		documents = make([]ragserver.Document, 0, 100)
	)

	for i, page := range response {
		pageNum := i + 1
		log.Printf("processing page %d/%d", pageNum, numPages)
		for _, aSentence := range tokenizer.Tokenize(page) {
			documents = append(documents, ragserver.Document{
				Text: aSentence.Text,
				Page: i + 1,
			})
		}
	}

	return documents, nil
}
