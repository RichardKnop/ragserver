package ragserver

import (
	"context"
	"fmt"
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
