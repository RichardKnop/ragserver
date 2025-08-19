package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/api"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

// Query the RAG server.
// (POST /query)
func (a *Adapter) Query(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		principal = a.principalFromRequest(r)
	)

	apiRequest := new(api.Query)
	if err := readRequestJSON(r, apiRequest); err != nil {
		renderJSONError(w, http.StatusBadRequest, err)
		return
	}

	var (
		query = ragserver.Query{
			Type: ragserver.QueryType(apiRequest.Type),
			Text: apiRequest.Content,
		}
		fileIDs []ragserver.FileID
	)

	for _, id := range apiRequest.FileIds {
		fileID, err := uuid.FromString(id.String())
		if err != nil {
			renderJSONError(w, http.StatusInternalServerError, err)
			return
		}
		fileIDs = append(fileIDs, ragserver.FileID{fileID})
	}

	if len(fileIDs) == 0 {
		renderJSONError(w, http.StatusBadRequest, fmt.Errorf("missing file IDs"))
		return
	}

	responses, err := a.ragServer.Generate(ctx, principal, query, fileIDs...)
	if err != nil {
		log.Printf("error generating response: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error generating response: %w", err))
		return
	}

	apiResponse := api.Responses{
		Responses: make([]api.ResponseItem, len(responses)),
	}
	for i, response := range responses {
		apiResponse.Responses[i] = api.ResponseItem{
			Type: api.ResponseItemType(response.Type),
			Text: response.Text,
		}
		if response.Type == ragserver.QueryTypeMetric {
			apiResponse.Responses[i].Metric = api.Float(response.Metric)
		}
	}

	renderJSON(w, apiResponse)
}
