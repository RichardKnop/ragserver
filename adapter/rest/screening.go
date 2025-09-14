package rest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/api"
)

// Create a screening
// (POST /screenings)
func (a *Adapter) CreateScreening(w http.ResponseWriter, r *http.Request) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), uploadTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	apiRequest := api.ScreeningParams{}
	if err := readRequestJSON(r, &apiRequest); err != nil {
		renderJSONError(w, http.StatusBadRequest, err)
		return
	}

	fileIDs, err := mapApiFileIDs(apiRequest.FileIds)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, err)
		return
	}

	aScreening, err := a.ragServer.CreateScreening(ctx, principal, ragserver.ScreeningParams{
		FileIDs:   fileIDs,
		Questions: mapApiQuestions(apiRequest.Questions),
	})
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error creating file: %w", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, mapScreening(aScreening))
}

func mapApiQuestions(apiQuestions []api.Question) []ragserver.Question {
	questions := make([]ragserver.Question, 0, len(apiQuestions))
	for _, apiQuestion := range apiQuestions {
		questions = append(questions, ragserver.Question{
			Type:    ragserver.QuestionType(apiQuestion.Type),
			Content: apiQuestion.Content,
		})
	}
	return questions
}

func mapScreening(screening *ragserver.Screening) api.Screening {
	return api.Screening{
		Id:            openapi_types.UUID(screening.ID.UUID[0:16]),
		Status:        api.ScreeningStatus(screening.Status),
		StatusMessage: api.String(screening.StatusMessage),
		Files:         mapFiles(screening.Files).Files,
		Questions:     mapQuestions(screening.Questions),
		CreatedAt:     screening.Created.T,
		UpdatedAt:     screening.Updated.T,
	}
}

func mapQuestions(questions []*ragserver.Question) []api.Question {
	apiQuestions := make([]api.Question, 0, len(questions))
	for _, question := range questions {
		apiQuestions = append(apiQuestions, mapQuestion(question))
	}
	return apiQuestions
}

func mapQuestion(question *ragserver.Question) api.Question {
	return api.Question{
		Type:    api.QuestionType(question.Type),
		Content: question.Content,
	}
}

// List screenings
// (GET /screenings)
func (a *Adapter) ListScreenings(w http.ResponseWriter, r *http.Request) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	screenings, err := a.ragServer.ListScreenings(ctx, principal)
	if err != nil {
		log.Printf("error listing screenings: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing screenings: %w", err))
		return
	}

	renderJSON(w, mapScreenings(screenings))
}

func mapScreenings(screenings []*ragserver.Screening) api.Screenings {
	apiResponse := api.Screenings{
		Screenings: make([]api.Screening, 0, len(screenings)),
	}
	for _, aScreening := range screenings {
		apiResponse.Screenings = append(apiResponse.Screenings, mapScreening(aScreening))
	}
	return apiResponse
}

// Get a single screening by ID
// (GET /screenings/{id})
func (a *Adapter) GetScreeningById(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	screeningID, err := uuid.FromString(id.String())
	if err != nil {
		log.Printf("invalid screening ID: %s", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid screening ID: %w", err))
		return
	}

	aScreening, err := a.ragServer.FindScreening(ctx, principal, ragserver.ScreeningID{UUID: screeningID})
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("screening not found"))
			return
		}
		log.Printf("error finding file: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error finding screening: %w", err))
		return
	}

	renderJSON(w, mapScreening(aScreening))
}
