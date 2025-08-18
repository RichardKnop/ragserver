package rest

import (
	"log"
	"net/http"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

func (a *Adapter) uploadFileHandler(w http.ResponseWriter, req *http.Request) {
	var (
		ctx       = req.Context()
		principal = a.principalFromRequest(req)
	)

	// Limit memory usage to 20MB, anythin over this limit will be stored in a temporary file.
	req.ParseMultipartForm(MaxFileSize)

	// Limit the size of the request body to prevent large uploads. This will return
	// io.MaxBytesError if the request body exceeds the limit while being read.
	req.Body = http.MaxBytesReader(w, req.Body, MaxFileSize)

	file, header, err := req.FormFile("file")
	if err != nil {
		log.Printf("error reading form file: %v", err)
		http.Error(w, "error reading form file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	aFile, err := a.ragServer.CreateFile(ctx, principal, file, header)
	if err != nil {
		log.Printf("error creating file: %v", err)
		http.Error(w, "error creating file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, aFile)
}

type FilesResponse struct {
	Files []*ragserver.File `json:"files"`
}

func (a *Adapter) listFileHandler(w http.ResponseWriter, req *http.Request) {
	var (
		ctx       = req.Context()
		principal = a.principalFromRequest(req)
	)

	files, err := a.ragServer.ListFiles(ctx, principal)
	if err != nil {
		log.Printf("error listing files: %v", err)
		http.Error(w, "error listing files", http.StatusInternalServerError)
		return
	}

	renderJSON(w, FilesResponse{
		Files: files,
	})
}
