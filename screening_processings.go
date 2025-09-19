package ragserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (rs *ragServer) ProcessScreenings(ctx context.Context) func() {
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

				total, err := rs.processScreenings(ctx)
				if err != nil {
					log.Println("error processing screenings:", err.Error())
				} else if total > 0 {
					log.Printf("processed %d screenings", total)
				}
			}
		}
	})

	return func() {
		wg.Wait()
		log.Println("Stopped processing screenings")
	}
}

const (
	processScreeningTimeout = 15 * time.Minute
)

func (rs *ragServer) processScreenings(ctx context.Context) (int, error) {
	var screenings []*Screening
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		// First let's check how many screenings are currently being processed
		// If there are too many, we won't pick up any new ones
		workersAvailable, err := rs.checkScreeningConcurrency(ctx)
		if err != nil {
			return err
		}
		if workersAvailable == 0 {
			return nil
		}

		// For now, let's only process a single screening at a time by each processing goroutine
		screenings, err = rs.store.ListScreenings(ctx, ScreeningFilter{
			Status: ScreeningStatusRequested,
			Lock:   true,
		}, rs.screeningPartial(), SortParams{
			Limit: workersAvailable,
			Order: SortOrderAsc,
			By:    `s."created"`,
		})
		if err != nil {
			return fmt.Errorf("list screenings: %w", err)
		}

		if len(screenings) == 0 {
			return nil
		}

		now := rs.now()
		for _, aScreening := range screenings {
			aScreening.Status = ScreeningStatusGenerating
			aScreening.Updated = now
			log.Printf("state change for screening: %s status: %s", aScreening.ID, aScreening.Status)
		}

		return rs.store.SaveScreenings(ctx, screenings...)
	}); err != nil {
		return 0, err
	}

	// TODO: process screenings in parallel?
	for _, aScreening := range screenings {
		processCtx, cancel := context.WithTimeout(ctx, processScreeningTimeout)
		defer cancel()
		if err := rs.processScreening(processCtx, aScreening); err != nil {
			if err := rs.processingScreeningFailed(ctx, aScreening, err); err != nil {
				log.Printf("error setting status to failed for screening: %s error %v", aScreening.ID, err)
			}
		}
	}

	// Now let's find screenings that have been processing for too long and mark them as failed
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		now := rs.now()

		screenings, err := rs.store.ListScreenings(ctx, ScreeningFilter{
			Status:            ScreeningStatusGenerating,
			LastUpdatedBefore: now.Add(-processScreeningTimeout - time.Minute),
		}, rs.screeningPartial(), SortParams{})
		if err != nil {
			return fmt.Errorf("list screenings to fail: %w", err)
		}

		for _, aScreening := range screenings {
			if err := aScreening.CompleteWithStatus(ScreeningStatusFailed, "timed out", now); err != nil {
				return fmt.Errorf("change status: %w", err)
			}
		}

		if err := rs.store.SaveScreenings(ctx, screenings...); err != nil {
			return fmt.Errorf("save screenings: %w", err)
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return len(screenings), nil
}

const maxConcurrentScreenings = 10

// checkScreeningConcurrency checks how many screenings are currently being processed,
// it is used to enforce a maximum number of concurrent screening processing jobs.
func (rs *ragServer) checkScreeningConcurrency(ctx context.Context) (int, error) {
	processing, err := rs.store.ListScreenings(ctx, ScreeningFilter{
		Status: ScreeningStatusGenerating,
	}, rs.screeningPartial(), SortParams{})
	if err != nil {
		return 0, fmt.Errorf("count screenings being processed: %w", err)
	}
	if len(processing) >= maxConcurrentScreenings {
		log.Printf("max concurrent screenings reached: %d", len(processing))
		return 0, nil
	}
	return maxConcurrentScreenings - len(processing), nil
}

func (rs *ragServer) processScreening(ctx context.Context, aScreening *Screening) error {
	for _, aQuestion := range aScreening.Questions {
		if err := rs.answwerQuestion(ctx, aQuestion, aScreening.FileIDs()...); err != nil {
			return err
		}
	}

	return rs.processingScreeningSucceeded(ctx, aScreening)
}

func (rs *ragServer) answwerQuestion(ctx context.Context, aQuestion *Question, fileIDs ...FileID) error {
	switch aQuestion.Type {
	case QuestionTypeText, QuestionTypeMetric, QuestionTypeBoolean:
	default:
		return fmt.Errorf("invalid question type: %s", aQuestion.Type)
	}

	_, err := rs.processedFilesFromIDs(ctx, fileIDs...)
	if err != nil {
		return err
	}

	log.Printf("generating answer for question: %s, file IDs: %v", aQuestion, fileIDs)

	// Embed the query contents.
	vector, err := rs.embedder.EmbedContent(ctx, aQuestion.Content)
	if err != nil {
		return fmt.Errorf("embedding query content: %v", err)
	}

	// Search weaviate to find the most relevant (closest in vector space)
	// documents to the query.
	documents, err := rs.retriever.SearchDocuments(ctx, DocumentFilter{
		Vector:  vector,
		FileIDs: fileIDs,
	}, 25)
	if err != nil {
		return fmt.Errorf("searching documents: %v", err)
	}

	if len(documents) == 0 {
		return fmt.Errorf("no documents found for question: %s", aQuestion)
	}

	log.Println("found documents:", len(documents))

	responses, err := rs.generative.Generate(ctx, *aQuestion, documents)
	if err != nil {
		return fmt.Errorf("calling generative model: %v", err)
	}

	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}

	jsonResponse, err := json.Marshal(responses[0])
	if err != nil {
		return fmt.Errorf("marshaling response: %v", err)
	}

	if err := rs.store.SaveAnswer(ctx, Answer{
		QuestionID: aQuestion.ID,
		Response:   string(jsonResponse),
		Created:    rs.now(),
	}); err != nil {
		return fmt.Errorf("saving answer: %w", err)
	}

	return nil
}

func (rs *ragServer) processingScreeningSucceeded(ctx context.Context, aScreening *Screening) error {
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := aScreening.CompleteWithStatus(ScreeningStatusCompleted, "", rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
		if err := rs.store.SaveScreenings(ctx, aScreening); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (rs *ragServer) processingScreeningFailed(ctx context.Context, aScreening *Screening, perr error) error {
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		if err := aScreening.CompleteWithStatus(ScreeningStatusFailed, perr.Error(), rs.now()); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
		if err := rs.store.SaveScreenings(ctx, aScreening); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
