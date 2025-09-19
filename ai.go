package ragserver

import (
	"context"
	"fmt"
	"log"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

type Vector []float32

type MetricValue struct {
	Value float64
	Unit  string
}

type BooleanValue bool

type Response struct {
	Text      string       `json:"text"`
	Metric    MetricValue  `json:"metric"`
	Boolean   BooleanValue `json:"boolean"`
	Documents []Document   `json:"documents"`
}

func (rs *ragServer) Generate(ctx context.Context, principal authz.Principal, question Question, fileIDs ...FileID) ([]Response, error) {
	switch question.Type {
	case QuestionTypeText, QuestionTypeMetric, QuestionTypeBoolean:
	default:
		return nil, fmt.Errorf("invalid question type: %s", question.Type)
	}

	_, err := rs.processedFilesFromIDs(ctx, fileIDs...)
	if err != nil {
		return nil, err
	}

	log.Printf("received question: %s, file IDs: %v", question, fileIDs)

	// Embed the query contents.
	vector, err := rs.embedder.EmbedContent(ctx, question.Content)
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
		return nil, fmt.Errorf("no documents found for question: %s", question)
	}

	log.Println("found documents:", len(documents))

	responses, err := rs.generative.Generate(ctx, question, documents)
	if err != nil {
		return nil, fmt.Errorf("calling generative model: %v", err)
	}

	return responses, nil
}

func (rs *ragServer) processedFilesFromIDs(ctx context.Context, ids ...FileID) ([]*File, error) {
	fileIDMap := map[FileID]struct{}{}
	for _, fileID := range ids {
		fileIDMap[fileID] = struct{}{}
	}

	if len(fileIDMap) < len(ids) {
		return nil, fmt.Errorf("duplicate file IDs provided")
	}

	files := make([]*File, 0, len(ids))

	// Check all file IDs exist in the database and that they have been processed.
	for _, fileID := range ids {
		aFile, err := rs.store.FindFile(ctx, fileID, rs.filePpartial())
		if err != nil {
			return nil, fmt.Errorf("error finding file: %v", err)
		}
		if aFile.Status != FileStatusProcessedSuccessfully {
			return nil, fmt.Errorf("file not processed: %s", fileID)
		}
		files = append(files, aFile)
	}

	return files, nil
}
