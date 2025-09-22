package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/api"
)

const defaultLimit = 100

// List file documents
// (GET /files/{id}/documents)
func (a *Adapter) ListFileDocuments(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params api.ListFileDocumentsParams) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	fileID, err := uuid.FromString(id.String())
	if err != nil {
		a.logger.Sugar().With("error", err).Error("invalid file ID")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid file ID: %w", err))
		return
	}

	if params.Limit != nil && api.FromInt(params.Limit) > 100 {
		renderJSONError(w, http.StatusBadRequest, fmt.Errorf("limit cannot be greater than 100"))
		return
	}

	limit := api.FromInt(params.Limit)
	if limit == 0 {
		limit = defaultLimit
	}

	documents, err := a.ragServer.ListFileDocuments(ctx, principal, ragserver.FileID{UUID: fileID}, ragserver.DocumentFilter{
		SimilarTo: api.FromString(params.SimilarTo),
	}, api.FromInt(params.Limit))
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("file documents not found"))
			return
		}
		a.logger.Sugar().With("error", err).Error("error listing file documents")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing file documents: %w", err))
		return
	}

	renderJSON(w, mapDocuments(documents))
}