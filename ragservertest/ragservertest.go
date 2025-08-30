package ragservertest

import (
	"time"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/RichardKnop/ragserver"
)

func New(seed int64, now time.Time) *DataGen {
	g := DataGen{
		Faker: gofakeit.New(seed),
		now:   now.UTC().Truncate(time.Millisecond),
	}

	return &g
}

type DataGen struct {
	*gofakeit.Faker
	now time.Time
}

type FileOption func(*ragserver.File)

func WithAuthorID(id ragserver.AuthorID) FileOption {
	return func(f *ragserver.File) {
		f.AuthorID = id
	}
}

func WithEmbedder(embedder string) FileOption {
	return func(f *ragserver.File) {
		f.Embedder = embedder
	}
}

func WithRetriever(retriever string) FileOption {
	return func(f *ragserver.File) {
		f.Retriever = retriever
	}
}

func WithStatus(status ragserver.FileStatus) FileOption {
	return func(f *ragserver.File) {
		f.Status = status
	}
}

func WithCreated(created time.Time) FileOption {
	return func(f *ragserver.File) {
		f.Created = ragserver.Time{T: created}
	}
}

func WithUpdated(updated time.Time) FileOption {
	return func(f *ragserver.File) {
		f.Updated = ragserver.Time{T: updated}
	}
}

var fileStates = []ragserver.FileStatus{
	ragserver.FileStatusUploaded,
	ragserver.FileStatusProcessing,
	ragserver.FileStatusProcessedSuccessfully,
	ragserver.FileStatusProcessingFailed,
}

func (g *DataGen) File(options ...FileOption) *ragserver.File {
	g.ShuffleAnySlice(fileStates)

	aFile := ragserver.File{
		ID:          ragserver.NewFileID(),
		AuthorID:    ragserver.NewAuthorID(),
		FileName:    g.Name() + ".pdf",
		ContentType: "application/pdf",
		Extension:   "pdf",
		Size:        g.Int64(),
		Hash:        g.LetterN(25),
		Embedder:    g.Name(),
		Retriever:   g.Name(),
		Location:    g.Word(),
		Status:      fileStates[0],
		Created:     ragserver.Time{T: g.now},
		Updated:     ragserver.Time{T: g.now},
	}

	for _, o := range options {
		o(&aFile)
	}

	return &aFile
}
