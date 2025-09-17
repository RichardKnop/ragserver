package store

import (
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
	"github.com/RichardKnop/ragserver/ragservertest"
)

var (
	testNow = time.Now().UTC()
	gen     = ragservertest.New(testNow.UnixNano(), testNow)
)

func (s *StoreTestSuite) TestFindFile() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		aFile = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileEmbedder("google-genai"),
			ragservertest.WithFileRetriever("redis"),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, aFile), "error saving file")

	s.Run("Find file without partial", func() {
		savedFile, err := s.adapter.FindFile(ctx, aFile.ID, authz.NilPartial)
		s.Require().NoError(err)
		s.Equal(aFile, savedFile)
	})

	s.Run("Find file with partial", func() {
		partial := authz.FilterBy("embedder", "google-genai").And("retriever", "weaviate")
		_, err := s.adapter.FindFile(ctx, aFile.ID, partial)
		s.Require().ErrorIs(err, ragserver.ErrNotFound)
	})
}

func (s *StoreTestSuite) TestSaveFiles_Upsert() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		now   = time.Now().UTC().Truncate(time.Millisecond)
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusUploaded),
			ragservertest.WithFileCreated(now),
			ragservertest.WithFileUpdated(now),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusProcessing),
			ragservertest.WithFileCreated(now),
			ragservertest.WithFileUpdated(now),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")

	// Save two files
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")

	savedFile1, err := s.adapter.FindFile(ctx, file1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file1, savedFile1)
	s.Equal(ragserver.FileStatusUploaded, savedFile1.Status)
	s.Equal(now, savedFile1.Updated.T)

	savedFile2, err := s.adapter.FindFile(ctx, file2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file2, savedFile2)
	s.Equal(ragserver.FileStatusProcessing, savedFile2.Status)
	s.Equal(now, savedFile1.Updated.T)

	// Let's save again to cause an upsert
	file1.Status = ragserver.FileStatusProcessing
	file1.Updated.T = file1.Updated.T.Add(1 * time.Minute)

	file2.Status = ragserver.FileStatusProcessingFailed
	file2.StatusMessage = "some error message"
	file2.Updated.T = file2.Updated.T.Add(2 * time.Minute)

	err = s.adapter.SaveFiles(ctx, file1, file2)
	s.Require().NoError(err)

	savedFile1, err = s.adapter.FindFile(ctx, file1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file1, savedFile1)
	s.Equal(ragserver.FileStatusProcessing, savedFile1.Status)
	s.Greater(savedFile1.Updated.T, now)

	savedFile2, err = s.adapter.FindFile(ctx, file2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file2, savedFile2)
	s.Equal(ragserver.FileStatusProcessingFailed, savedFile2.Status)
	s.Equal("some error message", savedFile2.StatusMessage)
	s.Greater(savedFile2.Updated.T, savedFile1.Updated.T)
}

func (s *StoreTestSuite) TestListFiles() {
	ctx, cancel := testContext()
	defer cancel()

	files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Empty(files)

	var (
		now   = time.Now().UTC().Truncate(time.Millisecond)
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusProcessing),
			ragservertest.WithFileCreated(now.Add(-1*time.Hour)),
			ragservertest.WithFileUpdated(now.Add(-1*time.Hour)),
			ragservertest.WithFileEmbedder("google-genai"),
			ragservertest.WithFileRetriever("weaviate"),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusProcessedSuccessfully),
			ragservertest.WithFileCreated(now),
			ragservertest.WithFileUpdated(now),
			ragservertest.WithFileEmbedder("google-genai"),
			ragservertest.WithFileRetriever("redis"),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")

	s.Run("List all files, no filter", func() {
		files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(files, 2)
		s.Contains(files, file1)
		s.Contains(files, file2)
	})

	s.Run("List all files, with limit", func() {
		files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial, ragserver.SortParams{Limit: 1})
		s.Require().NoError(err)
		s.Len(files, 1)
	})

	s.Run("Filter by embedder and retriever", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			Embedder:  "google-genai",
			Retriever: "weaviate",
		}, authz.NilPartial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})

	s.Run("Filter by status", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			Status: ragserver.FileStatusProcessedSuccessfully,
		}, authz.NilPartial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file2, files[0])
	})

	s.Run("Filter by last updated before", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			LastUpdatedBefore: ragserver.Time{T: now.Add(-time.Minute)},
		}, authz.NilPartial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})

	s.Run("List with a partial", func() {
		partial := authz.FilterBy("embedder", "google-genai").And("retriever", "weaviate")
		files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, partial, ragserver.SortParams{})
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})
}

func (s *StoreTestSuite) TestListFilesForProcessing() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		now   = time.Now().UTC().Truncate(time.Millisecond)
		file1 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusProcessing),
			ragservertest.WithFileCreated(now.Add(-1*time.Minute)),
			ragservertest.WithFileUpdated(now.Add(-1*time.Minute)),
		)
		file2 = gen.File(
			ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())),
			ragservertest.WithFileStatus(ragserver.FileStatusUploaded),
			ragservertest.WithFileCreated(now),
			ragservertest.WithFileUpdated(now),
		)
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, file1, file2), "error saving files")

	ids, err := s.adapter.ListFilesForProcessing(ctx, ragserver.Time{T: now}, authz.NilPartial, 10)
	s.Require().NoError(err)
	s.Len(ids, 1)
	s.Equal(file2.ID, ids[0])

	sameFile1, err := s.adapter.FindFile(ctx, file1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file1, sameFile1)

	updatedFile2, err := s.adapter.FindFile(ctx, file2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.NotEqual(file2, updatedFile2)
	s.Equal(ragserver.FileStatusProcessing, updatedFile2.Status)
}

func (s *StoreTestSuite) TestDeleteFiles() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		aFile = gen.File(ragservertest.WithFileAuthorID(ragserver.AuthorID(testPrincipal.ID())))
	)

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SaveFiles(ctx, aFile), "error saving file")

	files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Len(files, 1)

	err = s.adapter.DeleteFiles(ctx, aFile)
	s.Require().NoError(err)

	files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial, ragserver.SortParams{})
	s.Require().NoError(err)
	s.Len(files, 0)
}
