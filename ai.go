package ragserver

import (
	"context"
	"fmt"
	"log"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

type Vector []float32

type QueryType string

const (
	QueryTypeText    QueryType = "text"
	QueryTypeMetric  QueryType = "metric"
	QueryTypeBoolean QueryType = "boolean"
)

type Query struct {
	Type QueryType
	Text string
}

type MetricValue struct {
	Value float64
	Unit  string
}

type BooleanValue bool

type Response struct {
	Text      string
	Metric    MetricValue
	Boolean   BooleanValue
	Documents []Document
}

func (rs *ragServer) Generate(ctx context.Context, principal authz.Principal, query Query, fileIDs ...FileID) ([]Response, error) {
	switch query.Type {
	case QueryTypeText, QueryTypeMetric, QueryTypeBoolean:
	default:
		return nil, fmt.Errorf("invalid query type: %s", query.Type)
	}

	fileIDMap := map[FileID]struct{}{}
	for _, fileID := range fileIDs {
		fileIDMap[fileID] = struct{}{}
	}

	if len(fileIDMap) < len(fileIDs) {
		return nil, fmt.Errorf("duplicate file IDs provided")
	}

	log.Printf("received query: %s, file IDs: %v", query, fileIDs)

	// Check all file IDs exist in the database and that they have been processed.
	for _, fileID := range fileIDs {
		aFile, err := rs.store.FindFile(ctx, fileID, rs.partial())
		if err != nil {
			return nil, fmt.Errorf("error finding file: %v", err)
		}
		if aFile.Status != FileStatusProcessedSuccessfully {
			return nil, fmt.Errorf("file not processed: %s", fileID)
		}
	}

	// Embed the query contents.
	vector, err := rs.embedder.EmbedContent(ctx, query.Text)
	if err != nil {
		return nil, fmt.Errorf("embedding query content: %v", err)
	}

	// Search weaviate to find the most relevant (closest in vector space)
	// documents to the query.
	documents, err := rs.retriever.SearchDocuments(ctx, DocumentFilter{
		Vector:  vector,
		FileIDs: fileIDs,
	}, 25)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %v", err)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents found for query: %s", query)
	}

	log.Println("found documents:", len(documents))

	responses, err := rs.generative.Generate(ctx, query, documents)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}

	return responses, nil
}
