package redis

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/redis/go-redis/v9"

	"github.com/RichardKnop/ragserver"
)

func (a *Adapter) SaveDocuments(ctx context.Context, documents []ragserver.Document, vectors []ragserver.Vector) error {
	if len(documents) != len(vectors) {
		return fmt.Errorf("documents and vectors must have the same length")
	}

	for i, vector := range vectors {
		key := fmt.Sprintf("doc:%v", uuid.Must(uuid.NewV4()))
		fields, err := a.client.HSet(ctx,
			key,
			map[string]any{
				"content":   documents[i].Content,
				"file_id":   documents[i].FileID.String(),
				"page":      documents[i].Page,
				"embedding": floatsToBytes(vector),
			},
		).Result()
		if fields == 0 {
			return fmt.Errorf("no fields were added to redis")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Adapter) ListDocumentsByFileID(ctx context.Context, id ragserver.FileID) ([]ragserver.Document, error) {
	query := fmt.Sprintf("@file_id:{%s}", escapeUUID(id.UUID))

	results, err := a.client.FTSearchWithArgs(ctx,
		a.indexName,
		query,
		&redis.FTSearchOptions{
			Return: []redis.FTSearchReturn{
				{FieldName: "content"},
				{FieldName: "file_id"},
				{FieldName: "page"},
			},
			DialectVersion: a.dialectVersion,
			Limit:          100, // Override default limit of 10
		},
	).Result()
	if err != nil {
		return nil, err
	}

	return mapRedisDocuments(results.Docs)
}

func escapeUUID(u uuid.UUID) string {
	return strings.ReplaceAll(u.String(), "-", "\\-")
}

func (a *Adapter) SearchDocuments(ctx context.Context, filter ragserver.DocumentFilter, limit int) ([]ragserver.Document, error) {
	if filter.Vector == nil {
		return nil, fmt.Errorf("vector is required for searching documents")
	}

	ids := make([]string, 0, len(filter.FileIDs))
	for _, fileID := range filter.FileIDs {
		ids = append(ids, escapeUUID(fileID.UUID))
	}
	fileIDFilter := strings.Join(ids, "|")

	var query string
	if fileIDFilter != "" {
		query += fmt.Sprintf("(@file_id:{%s})", fileIDFilter)
	} else {
		query += "*"
	}
	query += fmt.Sprintf("=>[KNN %d @embedding $vec AS vector_distance]", limit)

	// The results are ordered according to the value of the vector_distance field,
	// with the lowest distance indicating the greatest similarity to the query.
	results, err := a.client.FTSearchWithArgs(ctx,
		a.indexName,
		query,
		&redis.FTSearchOptions{
			Return: []redis.FTSearchReturn{
				{FieldName: "vector_distance"},
				{FieldName: "content"},
				{FieldName: "file_id"},
				{FieldName: "page"},
			},
			DialectVersion: a.dialectVersion,
			Params: map[string]any{
				"vec": floatsToBytes(filter.Vector),
			},
			SortBy: []redis.FTSearchSortBy{{FieldName: "vector_distance", Asc: true}},
			Limit:  limit,
		},
	).Result()
	if err != nil {
		return nil, err
	}

	for _, doc := range results.Docs {
		fmt.Printf(
			"ID: %v, Distance:%v, Content:'%v'\n",
			doc.ID, doc.Fields["vector_distance"], doc.Fields["content"],
		)
	}

	return mapRedisDocuments(results.Docs)
}

func mapRedisDocuments(rds []redis.Document) ([]ragserver.Document, error) {
	documents := make([]ragserver.Document, 0, len(rds))

	for _, rd := range rds {
		aDocument, err := mapRedisDocument(rd)
		if err != nil {
			return nil, err
		}
		documents = append(documents, aDocument)
	}

	return documents, nil
}

func mapRedisDocument(rd redis.Document) (ragserver.Document, error) {
	_, ok := rd.Fields["content"]
	if !ok {
		return ragserver.Document{}, fmt.Errorf("missing content field in document")
	}

	page, err := strconv.Atoi(rd.Fields["page"])
	if err != nil {
		return ragserver.Document{}, fmt.Errorf("invalid page number: %v", err)
	}

	fileID, err := uuid.FromString(rd.Fields["file_id"])
	if err != nil {
		return ragserver.Document{}, fmt.Errorf("invalid file_id: %v", err)
	}

	return ragserver.Document{
		FileID:  ragserver.FileID{UUID: fileID},
		Content: rd.Fields["content"],
		Page:    page,
	}, nil
}

// helper function to convert []float32 to []byte
func floatsToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)

	for i, f := range fs {
		u := math.Float32bits(f)
		binary.NativeEndian.PutUint32(buf[i*4:], u)
	}

	return buf
}
