package ragserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
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
							log.Println("random jitter failed:", err.Error())
						}
						return
					}
				}

				total, err := rs.processFiles(ctx)
				if err != nil {
					log.Println("error processing files:", err.Error())
				} else if total > 0 {
					log.Printf("processed %d files", total)
				}
			}
		}
	})

	return wg.Wait
}

func jitter(ctx context.Context, jitterDuration time.Duration) error {
	select {
	case <-time.After(jitterDuration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rs *ragServer) processFiles(ctx context.Context) (int, error) {
	var files []*File
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		// TODO: add limit to only process N files at a time
		files, err = rs.store.ListFiles(ctx, FileFilter{
			Status: FileStatusUploaded,
		}, rs.partial())
		if err != nil {
			return fmt.Errorf("list files: %w", err)
		}

		if len(files) == 0 {
			return nil
		}

		now := rs.now()

		for _, aFile := range files {
			if err := aFile.ChangeStatus(FileStatusProcessing, "", now); err != nil {
				return fmt.Errorf("change status: %w", err)
			}
		}

		if err := rs.store.SaveFiles(ctx, files...); err != nil {
			return fmt.Errorf("save files: %w", err)
		}

		return nil
	}); err != nil {
		return 0, err
	}

	// TODO: process files in parallel?
	for _, aFile := range files {
		if err := rs.processFile(ctx, aFile); err != nil {
			if err := rs.processingFileFailed(ctx, aFile, err); err != nil {
				log.Printf("error setting status to failed for file: %s error %v", aFile.ID, err)
			}
		}
	}

	// TODO: clean up old files from disk?
	// TODO: fail files that have been processing for too long?

	return len(files), nil
}

func (rs *ragServer) processFile(ctx context.Context, aFile *File) error {
	content, err := os.Open(aFile.Location)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer content.Close()

	log.Printf("processing file: %s location: %s", aFile.ID, aFile.Location)

	switch aFile.ContentType {
	case "application/pdf":
		var err error
		documents, err := rs.extractor.Extract(ctx, content, rs.relevantTopics)
		if err != nil {
			return fmt.Errorf("error processing PDF file: %w", err)
		}
		for i := 0; i < len(documents); i++ {
			documents[i].FileID = aFile.ID
		}
		aFile.Documents = documents
	case "image/jpeg", "image/png":
		// client := gosseract.NewClient()
		// defer client.Close()

		// if err := client.SetImageFromBytes(fileBytes); err != nil {
		// 	log.Printf("client.SetImageFromBytes error: %v", err.Error())
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// }

		// text, err := client.Text()
		// if err != nil {
		// 	log.Printf("client.Text error: %v", err.Error())
		// 	http.Error(w, "file processing error", http.StatusInternalServerError)
		// 	return
		// }

		// log.Printf("file processed, text: %v", text)

		return fmt.Errorf("image file processing not implemented yet")
	}

	// Use the batch embedding API to embed all documents at once.
	vectors, err := rs.embedder.EmbedDocuments(ctx, aFile.Documents)
	if err != nil {
		return fmt.Errorf("error generating vectors: %v", err)
	}

	log.Printf("generated vectors: %d", len(vectors))

	if err := rs.retriever.SaveDocuments(ctx, aFile.Documents, vectors); err != nil {
		return fmt.Errorf("saving embeddings: %v", err)
	}

	return rs.processingSucceeded(ctx, aFile)
}

func (rs *ragServer) processingSucceeded(ctx context.Context, aFile *File) error {
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := aFile.ChangeStatus(FileStatusProcessedSuccessfully, "", rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
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
		if err := aFile.ChangeStatus(FileStatusProcessingFailed, perr.Error(), rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
		if err := rs.store.SaveFiles(ctx, aFile); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
