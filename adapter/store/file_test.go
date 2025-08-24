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

	aFile := &ragserver.File{
		ID:        ragserver.NewFileID(),
		FileName:  "test.pdf",
		MimeType:  "application/pdf",
		Extension: "pdf",
		Size:      123,
		Hash:      "abc123",
		Embedder:  "google-genai",
		Retriever: "redis",
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.adapter.FindFile(ctx, aFile.ID, authz.NilPartial)
	s.Require().ErrorIs(err, ragserver.ErrNotFound)

	err = s.adapter.SaveFiles(ctx, aFile)
	s.Require().NoError(err)

	savedFile, err := s.adapter.FindFile(ctx, aFile.ID, authz.NilPartial)
	s.Require().NoError(err)
	s.Equal(aFile, savedFile)

	// Try applying a partial
	partial := authz.FilterBy("embedder", "google-genai").And("retriever", "weaviate")
	_, err = s.adapter.FindFile(ctx, aFile.ID, partial)
	s.Require().ErrorIs(err, ragserver.ErrNotFound)
}

func (s *StoreTestSuite) TestListFiles() {
	ctx, cancel := testContext()
	defer cancel()

	files, err := s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial)
	s.Require().NoError(err)
	s.Empty(files)

	var (
		file1 = &ragserver.File{
			ID:        ragserver.NewFileID(),
			FileName:  "test1.pdf",
			MimeType:  "application/pdf",
			Extension: "pdf",
			Size:      123,
			Hash:      "abc123",
			Embedder:  "google-genai",
			Retriever: "weaviate",
			CreatedAt: time.Now().UTC(),
		}
		file2 = &ragserver.File{
			ID:        ragserver.NewFileID(),
			FileName:  "test2.pdf",
			MimeType:  "application/pdf",
			Extension: "pdf",
			Size:      123,
			Hash:      "def123",
			Embedder:  "google-genai",
			Retriever: "redis",
			CreatedAt: time.Now().UTC(),
		}
	)

	err = s.adapter.SaveFiles(ctx, file1, file2)
	s.Require().NoError(err)

	files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, authz.NilPartial)
	s.Require().NoError(err)
	s.Len(files, 2)
	s.Contains(files, file1)
	s.Contains(files, file2)

	// Try applying a filter
	files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{
		Embedder:  "google-genai",
		Retriever: "weaviate",
	}, authz.NilPartial)
	s.Require().NoError(err)
	s.Len(files, 1)
	s.Equal(file1, files[0])

	// Try applying a partial
	partial := authz.FilterBy("embedder", "google-genai").And("retriever", "weaviate")
	files, err = s.adapter.ListFiles(ctx, ragserver.FileFilter{}, partial)
	s.Require().NoError(err)
	s.Len(files, 1)
	s.Equal(file1, files[0])
}
