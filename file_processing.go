package ragserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	processInterval = 1 * time.Second
	maxJitter       = 100 * time.Millisecond
)

func (rs *ragServer) ProcessFiles(ctx context.Context) func() {
	var (
		ticker = time.NewTicker(processInterval - maxJitter/2)
		rand   = rand.New(rand.NewSource(time.Now().UnixNano()))
		wg     = new(sync.WaitGroup)
	)
	wg.Go(func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if maxJitter > 0 {
					jitterDuration := time.Duration(rand.Int63n(int64(maxJitter)))
					if err := jitter(ctx, jitterDuration); err != nil {
						if !errors.Is(err, context.Canceled) {
							rs.logger.Sugar().With("error", err).Error("random jitter failed")
						}
						return
					}
				}

				total, err := rs.processFiles(ctx)
				if err != nil {
					rs.logger.Sugar().With("error", err).Error("error processing files")
				} else if total > 0 {
					rs.logger.Sugar().Infof("processed %d files", total)
				}
			}
		}
	})

	return func() {
		wg.Wait()
		rs.logger.Info("Stopped processing files")
	}
}

func jitter(ctx context.Context, jitterDuration time.Duration) error {
	select {
	case <-time.After(jitterDuration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

const (
	processFileTimeout = 5 * time.Minute
)

func (rs *ragServer) processFiles(ctx context.Context) (int, error) {
	var files []*File
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		// First let's check how many screenings are currently being processed
		// If there are too many, we won't pick up any new ones
		workersAvailable, err := rs.checkFileConcurrency(ctx)
		if err != nil {
			return err
		}
		if workersAvailable == 0 {
			return nil
		}

		files, err = rs.store.ListFiles(ctx, FileFilter{
			Status: FileStatusUploaded,
			Lock:   true,
		}, rs.filePpartial(), SortParams{
			Limit: workersAvailable,
			Order: SortOrderAsc,
			By:    `f."created"`,
		})
		if err != nil {
			return fmt.Errorf("list files: %w", err)
		}

		if len(files) == 0 {
			return nil
		}

		now := rs.now()
		for _, aFile := range files {
			aFile.Status = FileStatusProcessing
			aFile.Updated = now
			rs.logger.Sugar().With("id", aFile.ID, "status", aFile.Status).Info("state change for file")
		}

		return rs.store.SaveFiles(ctx, files...)
	}); err != nil {
		return 0, err
	}

	// TODO: process files in parallel?
	for _, aFile := range files {
		processCtx, cancel := context.WithTimeout(ctx, processFileTimeout)
		defer cancel()
		if err := rs.processFile(processCtx, aFile); err != nil {
			if err := rs.processingFileFailed(ctx, aFile, err); err != nil {
				rs.logger.Sugar().With("id", aFile.ID, "error", err).Error("error setting status to failed for file")
			}
		}
	}

	// Now let's find files that have been processing for too long and mark them as failed
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		now := rs.now()

		files, err := rs.store.ListFiles(ctx, FileFilter{
			Status:            FileStatusProcessing,
			LastUpdatedBefore: now.Add(-processFileTimeout - time.Minute),
		}, rs.filePpartial(), SortParams{})
		if err != nil {
			return fmt.Errorf("list files to fail: %w", err)
		}

		for _, aFile := range files {
			if err := aFile.CompleteWithStatus(FileStatusProcessingFailed, "timed out", now); err != nil {
				return fmt.Errorf("change status: %w", err)
			}
			rs.logger.Sugar().With("id", aFile.ID, "status", aFile.Status).Info("state change for file")
		}

		if err := rs.store.SaveFiles(ctx, files...); err != nil {
			return fmt.Errorf("save files: %w", err)
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return len(files), nil
}

const maxConcurrentFiles = 10

// checkFileConcurrency checks how many files are currently being processed,
// it is used to enforce a maximum number of concurrent file processing jobs.
func (rs *ragServer) checkFileConcurrency(ctx context.Context) (int, error) {
	processing, err := rs.store.ListFiles(ctx, FileFilter{
		Status: FileStatusProcessing,
	}, rs.filePpartial(), SortParams{})
	if err != nil {
		return 0, fmt.Errorf("count files being processed: %w", err)
	}
	if len(processing) >= maxConcurrentFiles {
		rs.logger.Sugar().Infof("max concurrent files reached: %d", len(processing))
		return 0, nil
	}
	return maxConcurrentFiles - len(processing), nil
}

func (rs *ragServer) processFile(ctx context.Context, aFile *File) error {
	content, err := rs.filestorage.Read(aFile.Hash)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() {
		if err := rs.filestorage.Delete(aFile.Hash); err != nil {
			rs.logger.Sugar().With("hash", aFile.Hash, "error", err).Error("error removing file")
		}
		if err := content.Close(); err != nil {
			rs.logger.Sugar().With("hash", aFile.Hash, "error", err).Error("error closing file")
		}
	}()

	rs.logger.Sugar().With("id", aFile.ID, "hash", aFile.Hash).Info("processing file")

	switch aFile.ContentType {
	case "application/pdf":
		var err error
		documents, err := rs.extractor.Extract(ctx, content, rs.relevantTopics)
		if err != nil {
			return fmt.Errorf("error processing PDF file: %w", err)
		}
		for i := 0; i < len(documents); i++ {
			documents[i].FileID = aFile.ID
			documents[i] = documents[i].Sanitize()
		}
		aFile.Documents = documents
	case "image/jpeg", "image/png":
		// client := gosseract.NewClient()
		// defer client.Close()

		// if err := client.SetImageFromBytes(fileBytes); err != nil {
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// }

		// text, err := client.Text()
		// if err != nil {
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// 	return
		// }

		return fmt.Errorf("image file processing not implemented yet")
	}

	rs.logger.Sugar().Infof("extracted documents: %d", len(aFile.Documents))

	// Use the batch embedding API to embed all documents at once.
	vectors, err := rs.embedder.EmbedDocuments(ctx, aFile.Documents)
	if err != nil {
		return fmt.Errorf("error generating vectors: %v", err)
	}

	rs.logger.Sugar().Infof("generated vectors: %d", len(vectors))

	if err := rs.retriever.SaveDocuments(ctx, aFile.Documents, vectors); err != nil {
		return fmt.Errorf("saving embeddings: %v", err)
	}

	return rs.processingFileSucceeded(ctx, aFile)
}

func (rs *ragServer) processingFileSucceeded(ctx context.Context, aFile *File) error {
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := aFile.CompleteWithStatus(FileStatusProcessedSuccessfully, "", rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
		rs.logger.Sugar().With("id", aFile.ID, "status", aFile.Status).Info("state change for file")
		if err := rs.store.SaveFiles(ctx, aFile); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (rs *ragServer) processingFileFailed(ctx context.Context, aFile *File, perr error) error {
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := aFile.CompleteWithStatus(FileStatusProcessingFailed, perr.Error(), rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
		rs.logger.Sugar().With("id", aFile.ID, "status", aFile.Status).Info("state change for file")
		if err := rs.store.SaveFiles(ctx, aFile); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
