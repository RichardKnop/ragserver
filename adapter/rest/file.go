package rest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/api"
)

// TODO - implement a file lifecycle so UploadFile can return relatively quickly
// and the file is processed in the background. This will allow us to return a 202 Accepted
// response with a Location header pointing to the file resource, which can be polled for status.
const uploadTimeout = 300 * time.Second

// Upload a file and add documents extracted from it to the knowledge base
// (POST /files)
func (a *Adapter) UploadFile(w http.ResponseWriter, r *http.Request) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), uploadTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	// Limit memory usage to 20MB, anythin over this limit will be stored in a temporary file.
	r.ParseMultipartForm(ragserver.MaxFileSize)

	// Limit the size of the request body to prevent large uploads. This will return
	// io.MaxBytesError if the request body exceeds the limit while being read.
	r.Body = http.MaxBytesReader(w, r.Body, ragserver.MaxFileSize)

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

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, mapFile(aFile))
}

func mapFile(file *ragserver.File) api.File {
	return api.File{
		Id:            openapi_types.UUID(file.ID.UUID[0:16]),
		FileName:      file.FileName,
		ContentType:   file.ContentType,
		Extension:     file.Extension,
		Size:          file.Size,
		Hash:          file.Hash,
		Status:        api.FileStatus(file.Status),
		StatusMessage: file.StatusMessage,
		CreatedAt:     file.CreatedAt.T,
		UpdatedAt:     file.UpdatedAt.T,
	}
}

// List uploaded files
// (GET /files)
func (a *Adapter) ListFiles(w http.ResponseWriter, r *http.Request) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	files, err := a.ragServer.ListFiles(ctx, principal)
	if err != nil {
		log.Printf("error listing files: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing files: %w", err))
		return
	}

	renderJSON(w, mapFiles(files))
}

func mapFiles(files []*ragserver.File) api.Files {
	apiResponse := api.Files{
		Files: make([]api.File, 0, len(files)),
	}
	for _, file := range files {
		apiResponse.Files = append(apiResponse.Files, mapFile(file))
	}
	return apiResponse
}

// Get a single file by ID
// (GET /files/{id})
func (a *Adapter) GetFileById(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	fileID, err := uuid.FromString(id.String())
	if err != nil {
		log.Printf("invalid file ID: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid file ID: %w", err))
		return
	}

	aFile, err := a.ragServer.FindFile(ctx, principal, ragserver.FileID{UUID: fileID})
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("file not found"))
			return
		}
		log.Printf("error finding file: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error finding file: %w", err))
		return
	}

	renderJSON(w, mapFile(aFile))
}

// List file documents
// (GET /files/{id}/documents)
func (a *Adapter) ListFileDocuments(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	fileID, err := uuid.FromString(id.String())
	if err != nil {
		log.Printf("invalid file ID: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid file ID: %w", err))
		return
	}

	documents, err := a.ragServer.ListFileDocuments(ctx, principal, ragserver.FileID{UUID: fileID})
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("file documents not found"))
			return
		}
		log.Printf("error listing file documents: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing file documents: %w", err))
		return
	}

	renderJSON(w, mapDocuments(documents))
}

func mapDocument(document ragserver.Document) api.Document {
	return api.Document{
		Content: document.Content,
		Page:    int32(document.Page),
	}
}

func mapDocuments(documents []ragserver.Document) api.Documents {
	apiResponse := api.Documents{
		Documents: make([]api.Document, 0, len(documents)),
	}
	for _, doc := range documents {
		apiResponse.Documents = append(apiResponse.Documents, mapDocument(doc))
	}
	return apiResponse
}
