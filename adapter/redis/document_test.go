package redis

import (
	"math/rand/v2"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver"
)

func (s *RedisTestSuite) TestSearchDocuments() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		fileID1   = ragserver.FileID{UUID: uuid.Must(uuid.NewV4())}
		fileID2   = ragserver.FileID{UUID: uuid.Must(uuid.NewV4())}
		documents = []ragserver.Document{
			{
				Content: "This is a test document.",
				FileID:  fileID1,
				Page:    1,
			},
			{
				Content: "This is another test document.",
				FileID:  fileID1,
				Page:    2,
			},
			{
				Content: "This is a document from another file.",
				FileID:  fileID2,
				Page:    3,
			},
		}
		vectors = []ragserver.Vector{
			testVector(s.adapter.vectorDim, 0, 100),
			testVector(s.adapter.vectorDim, 0, 2),
			testVector(s.adapter.vectorDim, 0, 20),
		}
		searchVector = testVector(s.adapter.vectorDim, 0, 5)
	)

	err := s.adapter.SaveDocuments(ctx, documents, vectors)
	s.Require().NoError(err)

	results, err := s.adapter.SearchDocuments(
		ctx,
		ragserver.DocumentFilter{
			Vector:  searchVector,
			FileIDs: []ragserver.FileID{fileID1, fileID2},
		},
		25,
	)
	s.Require().NoError(err)
	s.Require().Len(results, 3)
	s.Equal(documents[1].Content, results[0].Content)
	s.Equal(documents[2].Content, results[1].Content)
	s.Equal(documents[0].Content, results[2].Content)
}

func (s *RedisTestSuite) TestListDocumentsByFileID() {
	ctx, cancel := testContext()
	defer cancel()

	var (
		fileID1   = ragserver.FileID{UUID: uuid.Must(uuid.NewV4())}
		fileID2   = ragserver.FileID{UUID: uuid.Must(uuid.NewV4())}
		documents = []ragserver.Document{
			{
				Content: "This is a test document.",
				FileID:  fileID1,
				Page:    1,
			},
			{
				Content: "This is another test document.",
				FileID:  fileID1,
				Page:    2,
			},
			{
				Content: "This is a document from another file.",
				FileID:  fileID2,
				Page:    3,
			},
		}
		vectors = []ragserver.Vector{
			testVector(s.adapter.vectorDim, 0, 100),
			testVector(s.adapter.vectorDim, 0, 2),
			testVector(s.adapter.vectorDim, 0, 20),
		}
	)

	err := s.adapter.SaveDocuments(ctx, documents, vectors)
	s.Require().NoError(err)

	results, err := s.adapter.ListDocumentsByFileID(ctx, fileID1)
	s.Require().NoError(err)
	s.Require().Len(results, 2)
	s.Contains(results, documents[0])
	s.Contains(results, documents[1])

	results, err = s.adapter.ListDocumentsByFileID(ctx, fileID2)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	s.Equal(documents[2].Content, results[0].Content)
}

func testVector(dim int, min, max float32) ragserver.Vector {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = min + rand.Float32()*(max-min)
	}
	return vec
}
