package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

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

	apiRequest := new(api.Question)
	if err := readRequestJSON(r, apiRequest); err != nil {
		renderJSONError(w, http.StatusBadRequest, err)
		return
	}

	fileIDs, err := mapOpenApiFileIDs(apiRequest.FileIds)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, err)
		return
	}

	if len(fileIDs) == 0 {
		renderJSONError(w, http.StatusBadRequest, fmt.Errorf("missing file IDs"))
		return
	}

	query := ragserver.Query{
		Type: ragserver.QueryType(apiRequest.Type),
		Text: apiRequest.Content,
	}

	responses, err := a.ragServer.Generate(ctx, principal, query, fileIDs...)
	if err != nil {
		log.Printf("error generating response: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error generating response: %w", err))
		return
	}

	renderJSON(w, mapResponse(*apiRequest, responses))
}

func mapOpenApiFileIDs(ids []openapi_types.UUID) ([]ragserver.FileID, error) {
	fileIDs := make([]ragserver.FileID, 0, len(ids))
	for _, id := range ids {
		fileID, err := uuid.FromString(id.String())
		if err != nil {
			return nil, err
		}
		fileIDs = append(fileIDs, ragserver.FileID{UUID: fileID})
	}
	return fileIDs, nil
}

func mapResponse(question api.Question, responses []ragserver.Response) api.Response {
	apiResponse := api.Response{
		Question: question,
		Answers:  make([]api.Answer, 0, len(responses)),
	}
	for _, response := range responses {
		answerItem := api.Answer{
			Text: response.Text,
		}
		if question.Type == api.QuestionType(ragserver.QueryTypeMetric) {
			answerItem.Metric = &api.MetricValue{
				Value: response.Metric.Value,
				Unit:  api.String(response.Metric.Unit),
			}
		}
		apiResponse.Answers = append(apiResponse.Answers, answerItem)
	}
	return apiResponse
}
