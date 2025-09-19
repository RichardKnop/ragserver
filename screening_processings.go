package ragserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	processScreeningTimeout = 15 * time.Minute
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

func (rs *ragServer) processScreenings(ctx context.Context) (int, error) {
	var screenings []*Screening
	if err := rs.store.Transactional(ctx, &sql.TxOptions{}, func(ctx context.Context) error {
		var err error
		// For now, let's only process a single screening at a time by each processing goroutine
		screenings, err = rs.store.ListScreenings(ctx, ScreeningFilter{}, rs.screeningPartial(), SortParams{
			Limit: 1,
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
			LastUpdatedBefore: now.Add(-processScreeningTimeout),
		}, rs.filePpartial(), SortParams{})
		if err != nil {
			return fmt.Errorf("list screenings: %w", err)
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

func (rs *ragServer) processScreening(ctx context.Context, aScreening *Screening) error {
	// TODO - implement
	return fmt.Errorf("not implemented")
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
