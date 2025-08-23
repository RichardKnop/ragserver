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

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

func (a *Adapter) SaveDocuments(ctx context.Context, documents []ragserver.Document, vectors []ragserver.Vector) error {
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

func (a *Adapter) ListDocuments(ctx context.Context, filter ragserver.DocumentFilter) ([]ragserver.Document, error) {
	ids := make([]string, 0, len(filter.FileIDs))
	for _, fileID := range filter.FileIDs {
		ids = append(ids, escapeUUID(fileID.UUID))
	}
	fileIDFilter := strings.Join(ids, "|")

	var query string
	if fileIDFilter != "" {
		query += fmt.Sprintf("@file_id:{%s}", fileIDFilter)
	}

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

func (a *Adapter) SearchDocuments(ctx context.Context, vector ragserver.Vector, limit int, fileIDs ...ragserver.FileID) ([]ragserver.Document, error) {
	ids := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		ids = append(ids, escapeUUID(fileID.UUID))
	}
	fileIDFilter := strings.Join(ids, "|")

	var query string
	if fileIDFilter != "" {
		query += fmt.Sprintf("(@file_id:{%s})=>", fileIDFilter)
	} else {
		query += "*=>"
	}
	query += ">[KNN 3 @embedding $vec AS vector_distance]"

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
				"vec": floatsToBytes(vector),
			},
			Limit: limit,
		},
	).Result()
	if err != nil {
		return nil, err
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
