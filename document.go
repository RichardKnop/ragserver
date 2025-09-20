package ragserver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

type Vector []float32

type Document struct {
	FileID  FileID `json:"file_id"`
	Content string `json:"content"`
	Page    int    `json:"page"`
}

type DocumentFilter struct {
	Vector  Vector
	FileIDs []FileID
}

type Topic struct {
	Name     string
	Keywords []string
}

func (d Document) Sanitize() Document {
	d.Content = strings.TrimSpace(d.Content)
	d.Content = strings.Join(strings.Fields(d.Content), " ")
	return d
}

type RelevantTopics []Topic

func (rt RelevantTopics) IsRelevant(content string) (Topic, bool) {
	for len(rt) == 0 {
		return Topic{}, false
	}

	for _, topic := range rt {
		for _, keyword := range topic.Keywords {
			if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				return topic, true
			}
		}
	}

	return Topic{}, false
}

func (rs *ragServer) ListFileDocuments(ctx context.Context, principal authz.Principal, id FileID) ([]Document, error) {
	var documents []Document
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		_, err := rs.store.FindFile(ctx, id, rs.filePpartial())
		if err != nil {
			return err
		}

		documents, err = rs.retriever.ListDocumentsByFileID(ctx, id)
		if err != nil {
			return fmt.Errorf("list documents by file ID: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return documents, nil
}

// MatchSnippetsToDocuments tries to match snippets to documents by exact match or by containment.
// It returns matched documents and remaining snippets that could not be matched to any document.
func MatchSnippetsToDocuments(possibleSnippets []string, documents []Document) ([]Document, []string) {
	var (
		snippets         = make([]string, 0, len(possibleSnippets))
		matchedDocuments = make([]Document, 0, len(documents))
	)

	// First sanitize snippets. It is not always possible to force LLM to always return snippets
	// exactly matching the documents, so we need to be a bit flexible.
	for _, possibleSnippet := range possibleSnippets {
		// Sometimes the model returns multiple snippets separated by new lines as one snippet,
		// so we need to split them and treat each one individually.
		for _, aSnippet := range strings.Split(possibleSnippet, "\n") {
			if strings.TrimSpace(aSnippet) == "" {
				continue
			}
			snippets = append(snippets, strings.TrimSpace(aSnippet))
		}
	}

	for _, aDocument := range documents {
		if len(snippets) == 0 {
			break
		}
		for i, aSnippet := range snippets {
			if aSnippet == aDocument.Content || strings.Contains(aDocument.Content, aSnippet) {
				matchedDocuments = append(matchedDocuments, aDocument)
				if len(snippets) == 1 {
					snippets = nil
					break
				}
				snippets = append(snippets[:i], snippets[i+1:]...)
				break
			}
		}
	}

	if len(matchedDocuments) == 0 {
		return nil, snippets
	}

	return matchedDocuments, snippets
}
