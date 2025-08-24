package ragserver

import (
	"context"
	"database/sql"
	"strings"

	"github.com/RichardKnop/ragserver/internal/pkg/authz"
)

type Document struct {
	FileID  FileID
	Content string
	Page    int
}

type DocumentFilter struct {
	Vector  Vector
	FileIDs []FileID
}

type Topic struct {
	Name     string
	Keywords []string
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
		_, err := rs.store.FindFile(ctx, id, rs.partial())
		if err != nil {
			return err
		}

		documents, err = rs.retriever.ListDocumentsByFileID(ctx, id)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return documents, nil
}
