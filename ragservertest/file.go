package ragservertest

import (
	"time"

	"github.com/RichardKnop/ragserver"
)

type FileOption func(*ragserver.File)

func WithFileAuthorID(id ragserver.AuthorID) FileOption {
	return func(f *ragserver.File) {
		f.AuthorID = id
	}
}

func WithFileEmbedder(embedder string) FileOption {
	return func(f *ragserver.File) {
		f.Embedder = embedder
	}
}

func WithFileRetriever(retriever string) FileOption {
	return func(f *ragserver.File) {
		f.Retriever = retriever
	}
}

func WithFileStatus(status ragserver.FileStatus) FileOption {
	return func(f *ragserver.File) {
		f.Status = status
	}
}

func WithFileCreated(created time.Time) FileOption {
	return func(f *ragserver.File) {
		f.Created = created
	}
}

func WithFileUpdated(updated time.Time) FileOption {
	return func(f *ragserver.File) {
		f.Updated = updated
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
		Status:      fileStates[0],
		Created:     g.now,
		Updated:     g.now,
	}

	for _, o := range options {
		o(&aFile)
	}

	return &aFile
}
