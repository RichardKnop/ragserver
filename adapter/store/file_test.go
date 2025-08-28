package store

import (
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/pkg/authz"
)

func (s *StoreTestSuite) TestFindFile() {
	ctx, cancel := testContext()
	defer cancel()

	now := time.Now().UTC().Truncate(time.Millisecond)
	aFile := &ragserver.File{
		ID:          ragserver.NewFileID(),
		FileName:    "test.pdf",
		ContentType: "application/pdf",
		Extension:   "pdf",
		Size:        123,
		Hash:        "abc123",
		Embedder:    "google-genai",
		Retriever:   "redis",
		Location:    "some/location",
		Status:      ragserver.FileStatusUploaded,
		CreatedAt:   ragserver.Time{T: now},
		UpdatedAt:   ragserver.Time{T: now},
	}

	_, err := s.adapter.FindFile(ctx, aFile.ID, authz.NilPartial)
	s.Require().ErrorIs(err, ragserver.ErrNotFound)

	err = s.adapter.SaveFiles(ctx, aFile)
	s.Require().NoError(err)

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

func (s *StoreTestSuite) TestSaveFile_Upsert() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		now   = time.Now().UTC().Truncate(time.Millisecond)
		file1 = &ragserver.File{
			ID:          ragserver.NewFileID(),
			FileName:    "test1.pdf",
			ContentType: "application/pdf",
			Extension:   "pdf",
			Size:        123,
			Hash:        "abc123",
			Embedder:    "google-genai",
			Retriever:   "weaviate",
			Location:    "some/location1",
			Status:      ragserver.FileStatusUploaded,
			CreatedAt:   ragserver.Time{T: now},
			UpdatedAt:   ragserver.Time{T: now},
		}
		file2 = &ragserver.File{
			ID:          ragserver.NewFileID(),
			FileName:    "test2.pdf",
			ContentType: "application/pdf",
			Extension:   "pdf",
			Size:        123,
			Hash:        "def123",
			Embedder:    "google-genai",
			Retriever:   "redis",
			Location:    "some/location2",
			Status:      ragserver.FileStatusProcessing,
			CreatedAt:   ragserver.Time{T: now},
			UpdatedAt:   ragserver.Time{T: now},
		}
	)

	// Save two files
	err := s.adapter.SaveFiles(ctx, file1, file2)
	s.Require().NoError(err)

	savedFile1, err := s.adapter.FindFile(ctx, file1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file1, savedFile1)
	s.Equal(ragserver.FileStatusUploaded, savedFile1.Status)
	s.Equal(now, savedFile1.UpdatedAt.T)

	savedFile2, err := s.adapter.FindFile(ctx, file2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file2, savedFile2)
	s.Equal(ragserver.FileStatusProcessing, savedFile2.Status)
	s.Equal(now, savedFile1.UpdatedAt.T)

	// Let's save again to cause an upsert
	file1.Status = ragserver.FileStatusProcessing
	file1.UpdatedAt.T = file1.UpdatedAt.T.Add(1 * time.Minute)

	file2.Status = ragserver.FileStatusProcessingFailed
	file2.StatusMessage = "some error message"
	file2.UpdatedAt.T = file2.UpdatedAt.T.Add(1 * time.Minute)

	err = s.adapter.SaveFiles(ctx, file1, file2)
	s.Require().NoError(err)

	savedFile1, err = s.adapter.FindFile(ctx, file1.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file1, savedFile1)
	s.Equal(ragserver.FileStatusProcessing, savedFile1.Status)
	s.Greater(savedFile1.UpdatedAt.T, now)

	savedFile2, err = s.adapter.FindFile(ctx, file2.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(file2, savedFile2)
	s.Equal(ragserver.FileStatusProcessingFailed, savedFile2.Status)
	s.Equal("some error message", savedFile2.StatusMessage)
	s.Greater(savedFile2.UpdatedAt.T, now)
}

func (s *StoreTestSuite) TestListFiles() {
	ctx, cancel := testContext()
	defer cancel()

	files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial)
	s.Require().NoError(err)
	s.Empty(files)

	var (
		now   = time.Now().UTC().Truncate(time.Millisecond)
		file1 = &ragserver.File{
			ID:          ragserver.NewFileID(),
			FileName:    "test1.pdf",
			ContentType: "application/pdf",
			Extension:   "pdf",
			Size:        123,
			Hash:        "abc123",
			Embedder:    "google-genai",
			Retriever:   "weaviate",
			Location:    "some/location1",
			Status:      ragserver.FileStatusProcessing,
			CreatedAt:   ragserver.Time{T: now.Add(-1 * time.Hour)},
			UpdatedAt:   ragserver.Time{T: now.Add(-1 * time.Hour)},
		}
		file2 = &ragserver.File{
			ID:          ragserver.NewFileID(),
			FileName:    "test2.pdf",
			ContentType: "application/pdf",
			Extension:   "pdf",
			Size:        123,
			Hash:        "def123",
			Embedder:    "google-genai",
			Retriever:   "redis",
			Location:    "some/location2",
			Status:      ragserver.FileStatusProcessedSuccessfully,
			CreatedAt:   ragserver.Time{T: now},
			UpdatedAt:   ragserver.Time{T: now},
		}
	)

	err = s.adapter.SaveFiles(ctx, file1, file2)
	s.Require().NoError(err)

	s.Run("List all files, no filter", func() {
		files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial)
		s.Require().NoError(err)
		s.Len(files, 2)
		s.Contains(files, file1)
		s.Contains(files, file2)
	})

	s.Run("Filter by embedder and retriever", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			Embedder:  "google-genai",
			Retriever: "weaviate",
		}, authz.NilPartial)
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})

	s.Run("Filter by status", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			Status: ragserver.FileStatusProcessedSuccessfully,
		}, authz.NilPartial)
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file2, files[0])
	})

	s.Run("Filter by last updated before", func() {
		files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{
			LastUpdatedBefore: ragserver.Time{T: now.Add(-time.Minute)},
		}, authz.NilPartial)
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})

	s.Run("List with a partial", func() {
		partial := authz.FilterBy("embedder", "google-genai").And("retriever", "weaviate")
		files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, partial)
		s.Require().NoError(err)
		s.Len(files, 1)
		s.Equal(file1, files[0])
	})
}
