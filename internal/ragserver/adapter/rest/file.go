package rest

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/RichardKnop/ragserver/api"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

const (
	MB          = 1 << 20
	MaxFileSize = 20 * MB
)

// Upload a file and add documents extracted from it to the knowledge base
// (POST /files)
func (a *Adapter) UploadFile(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		principal = a.principalFromRequest(r)
	)

	// Limit memory usage to 20MB, anythin over this limit will be stored in a temporary file.
	r.ParseMultipartForm(MaxFileSize)

	// Limit the size of the request body to prevent large uploads. This will return
	// io.MaxBytesError if the request body exceeds the limit while being read.
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)

	file, header, err := r.FormFile("file")
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error reading file from request: %w", err))
		return
	}
	defer file.Close()

	aFile, err := a.ragServer.CreateFile(ctx, principal, file, header)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error creating file: %w", err))
		return
	}

	apiResponse := mapFile(aFile)

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, apiResponse)
}

func mapFile(file *ragserver.File) api.File {
	return api.File{
		Id:        file.ID.String(),
		FileName:  file.FileName,
		MimeType:  file.MimeType,
		Extension: file.Extension,
		Size:      file.Size,
		CreatedAt: file.CreatedAt,
	}
}

// List uploaded files
// (GET /files)
func (a *Adapter) ListFiles(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		principal = a.principalFromRequest(r)
	)

	files, err := a.ragServer.ListFiles(ctx, principal)
	if err != nil {
		log.Printf("error listing files: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing files: %w", err))
		return
	}

	apiResponse := api.Files{
		Files: make([]api.File, 0, len(files)),
	}
	for _, file := range files {
		apiResponse.Files = append(apiResponse.Files, mapFile(file))
	}

	renderJSON(w, apiResponse)
}

// Get a single file by ID
// (GET /files/{id})
func (a *Adapter) GetFileById(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var (
		ctx       = r.Context()
		principal = a.principalFromRequest(r)
	)

	fileID, err := uuid.FromString(id.String())
	if err != nil {
		log.Printf("invalid file ID: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid file ID: %w", err))
		return
	}

	aFile, err := a.ragServer.FindFile(ctx, principal, ragserver.FileID{fileID})
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("file not found"))
			return
		}
		log.Printf("error finding file: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error finding file: %w", err))
		return
	}

	apiResponse := mapFile(aFile)

	renderJSON(w, apiResponse)
}
