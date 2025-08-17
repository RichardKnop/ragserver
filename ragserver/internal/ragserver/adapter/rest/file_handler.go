package rest

import (
	"log"
	"net/http"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

func (a *Adapter) uploadFileHandler(w http.ResponseWriter, req *http.Request) {
	// Limit memory usage to 20MB, anythin over this limit will be stored in a temporary file.
	req.ParseMultipartForm(MaxFileSize)

	// Limit the size of the request body to prevent large uploads. This will return
	// io.MaxBytesError if the request body exceeds the limit while being read.
	req.Body = http.MaxBytesReader(w, req.Body, MaxFileSize)

	file, header, err := req.FormFile("file")
	if err != nil {
		log.Printf("error reading form file: %v", err)
		http.Error(w, "error reading form file", http.StatusInternalServerError)
	}
	defer file.Close()

	aFile, err := a.ragServer.CreateFile(req.Context(), file, header)
	if err != nil {
		log.Printf("error creating file: %v", err)
		http.Error(w, "error creating file", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, aFile)
}
