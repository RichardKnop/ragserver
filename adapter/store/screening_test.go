package store

import (
	"time"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
	"github.com/RichardKnop/ragserver/ragservertest"
)

func (s *StoreTestSuite) TestFindScreening() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		screeningID = ragserver.NewScreeningID()
		question1   = gen.Question(
			ragservertest.WithQuestionAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithQuestionScreeningID(screeningID),
		)
		question2 = gen.Question(
			ragservertest.WithQuestionAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithQuestionScreeningID(screeningID),
		)
		aScreening = gen.Screening(
			ragservertest.WithScreeningID(screeningID),
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(file1, file2),
			ragservertest.WithScreeningQuestions(question1, question2),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")
	s.Require().NoError(s.adapter.SaveScreenings(ctx, aScreening), "error saving screening")
	s.Require().NoError(s.adapter.SaveScreeningFiles(ctx, aScreening), "error saving screening files")
	s.Require().NoError(s.adapter.SaveScreeningQuestions(ctx, aScreening), "error saving screening questions")

	s.Run("Find screening without partial", func() {
		savedScreening, err := s.adapter.FindScreening(ctx, aScreening.ID, authz.NilPartial)
		s.Require().NoError(err)
		s.Equal(aScreening, savedScreening)
	})
}

func (s *StoreTestSuite) TestSaveScreenings_Upsert() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		now   = time.Now().UTC()
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		screening1 = gen.Screening(
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(file1),
			ragservertest.WithScreeningStatus(ragserver.ScreeningStatusRequested),
			ragservertest.WithScreeningCreated(now),
			ragservertest.WithScreeningUpdated(now),
		)
		screening2 = gen.Screening(
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(file2),
			ragservertest.WithScreeningStatus(ragserver.ScreeningStatusRequested),
			ragservertest.WithScreeningCreated(now),
			ragservertest.WithScreeningUpdated(now),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")
	s.Require().NoError(s.adapter.SaveScreenings(ctx, screening1, screening2), "error saving screening")
	s.Require().NoError(s.adapter.SaveScreeningFiles(ctx, screening1, screening2), "error saving screening files")

	savedScreening1, err := s.adapter.FindScreening(ctx, screening1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(screening1, savedScreening1)
	s.Equal(ragserver.ScreeningStatusRequested, savedScreening1.Status)
	s.Equal(now, savedScreening1.Updated)

	savedScreening2, err := s.adapter.FindScreening(ctx, screening2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(screening2, savedScreening2)
	s.Equal(ragserver.ScreeningStatusRequested, savedScreening2.Status)
	s.Equal(now, savedScreening2.Updated)

	// Let's save again to cause an upsert
	screening1.Status = ragserver.ScreeningStatusGenerating
	screening1.Updated = screening1.Updated.Add(1 * time.Minute).UTC()

	screening2.Status = ragserver.ScreeningStatusFailed
	screening2.StatusMessage = "some error message"
	screening2.Updated = screening2.Updated.Add(2 * time.Minute).UTC()

	err = s.adapter.SaveScreenings(ctx, screening1, screening2)
	s.Require().NoError(err)

	savedScreening1, err = s.adapter.FindScreening(ctx, screening1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(screening1, savedScreening1)
	s.Equal(ragserver.ScreeningStatusGenerating, savedScreening1.Status)
	s.Greater(savedScreening1.Updated, now)

	savedScreening2, err = s.adapter.FindScreening(ctx, screening2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(screening2, savedScreening2)
	s.Equal(ragserver.ScreeningStatusFailed, savedScreening2.Status)
	s.Equal("some error message", savedScreening2.StatusMessage)
	s.Greater(savedScreening2.Updated, savedScreening1.Updated)
}

func (s *StoreTestSuite) TestListScreenings() {
	ctx, cancel := testContext()
	defer cancel()

	screenings, err := s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Empty(screenings)

	var (
		now   = time.Now().UTC()
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		screening1 = gen.Screening(
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(file1),
			ragservertest.WithScreeningStatus(ragserver.ScreeningStatusCompleted),
			ragservertest.WithScreeningCreated(now.Add(-1*time.Hour).UTC()),
			ragservertest.WithScreeningUpdated(now.Add(-1*time.Hour).UTC()),
		)
		screening2 = gen.Screening(
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(file2),
			ragservertest.WithScreeningStatus(ragserver.ScreeningStatusRequested),
			ragservertest.WithScreeningCreated(now),
			ragservertest.WithScreeningUpdated(now),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")
	s.Require().NoError(s.adapter.SaveScreenings(ctx, screening1, screening2), "error saving screening")
	s.Require().NoError(s.adapter.SaveScreeningFiles(ctx, screening1, screening2), "error saving screening files")

	s.Run("List all screenings, no filter", func() {
		screenings, err = s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{}, authz.NilPartial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(screenings, 2)
		s.Equal(screenings[0], screening2)
		s.Equal(screenings[1], screening1)
	})

	s.Run("List all screenings, with limit", func() {
		screenings, err = s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{}, authz.NilPartial, ragserver.SortParams{Limit: 1})
		s.Require().NoError(err)
		s.Len(screenings, 1)
	})

	s.Run("For update skip locked", func() {
		files, err := s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{
			Status: ragserver.ScreeningStatusRequested,
			Lock:   true,
		}, authz.NilPartial, ragserver.SortParams{
			Limit: 1,
			Order: ragserver.SortOrderAsc,
			By:    `s."created"`,
		})
		s.Require().NoError(err)
		s.Len(files, 1)
	})
}

func (s *StoreTestSuite) TestDeleteScreenings() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		aFile = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
		)
		aScreening = gen.Screening(
			ragservertest.WithScreeningAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithScreeningFiles(aFile),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, aFile), "error saving files")
	s.Require().NoError(s.adapter.SaveScreenings(ctx, aScreening), "error saving screening")
	s.Require().NoError(s.adapter.SaveScreeningFiles(ctx, aScreening), "error saving screening files")

	screenings, err := s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Len(screenings, 1)

	err = s.adapter.DeleteScreenings(ctx, aScreening)
	s.Require().NoError(err)

	screenings, err = s.adapter.ListScreenings(ctx, ragserver.ScreeningFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Len(screenings, 0)
}
