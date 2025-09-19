package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	apiScreening, err := mapScreening(aScreening)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error mapping screening: %w", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	renderJSON(w, apiScreening)
}

func mapApiFileIDs(ids []openapi_types.UUID) ([]ragserver.FileID, error) {
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

func mapApiQuestions(apiQuestions []api.QuestionParams) []ragserver.Question {
	questions := make([]ragserver.Question, 0, len(apiQuestions))
	for _, apiQuestion := range apiQuestions {
		questions = append(questions, ragserver.Question{
			Type:    ragserver.QuestionType(apiQuestion.Type),
			Content: apiQuestion.Content,
		})
	}
	return questions
}

func mapScreening(screening *ragserver.Screening) (api.Screening, error) {
	apiScreening := api.Screening{
		Id:            openapi_types.UUID(screening.ID.UUID[0:16]),
		Status:        api.ScreeningStatus(screening.Status),
		StatusMessage: api.String(screening.StatusMessage),
		Files:         mapFiles(screening.Files).Files,
		Questions:     mapQuestions(screening.Questions),
		CreatedAt:     screening.Created,
		UpdatedAt:     screening.Updated,
	}

	var err error
	apiScreening.Answers, err = mapAnswers(screening.Questions, screening.Answers)
	if err != nil {
		return api.Screening{}, err
	}

	return apiScreening, nil
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
		Id:      openapi_types.UUID(question.ID.UUID[0:16]),
		Type:    api.QuestionType(question.Type),
		Content: question.Content,
	}
}

func mapAnswers(questions []*ragserver.Question, answers []ragserver.Answer) ([]api.Answer, error) {
	questionMap := map[ragserver.QuestionID]*ragserver.Question{}
	for _, question := range questions {
		questionMap[question.ID] = question
	}

	apiAnswers := make([]api.Answer, 0, len(answers))
	for _, answer := range answers {
		if _, ok := questionMap[answer.QuestionID]; !ok {
			return nil, fmt.Errorf("question not found for answer: %s", answer.QuestionID)
		}
		apiAnswer, err := mapAnswer(questionMap[answer.QuestionID], answer)
		if err != nil {
			return nil, err
		}
		apiAnswers = append(apiAnswers, apiAnswer)
	}
	return apiAnswers, nil
}

func mapAnswer(question *ragserver.Question, answer ragserver.Answer) (api.Answer, error) {
	var response = new(ragserver.Response)
	if err := json.Unmarshal([]byte(answer.Response), response); err != nil {
		return api.Answer{}, fmt.Errorf("error unmarshaling answer response: %w", err)
	}

	apiAnswer := api.Answer{
		QuestionId: openapi_types.UUID(question.ID.UUID[0:16]),
		Text:       string(response.Text),
	}

	if question.Type == ragserver.QuestionTypeMetric {
		apiAnswer.Metric = &api.MetricValue{
			Value: response.Metric.Value,
			Unit:  api.String(response.Metric.Unit),
		}
	}
	if question.Type == ragserver.QuestionTypeBoolean {
		apiAnswer.Boolean = api.Boolean(bool(response.Boolean))
	}
	apiAnswer.Evidence = make([]api.Evidence, 0, len(response.Documents))
	for _, doc := range response.Documents {
		apiAnswer.Evidence = append(apiAnswer.Evidence, api.Evidence{
			FileId: openapi_types.UUID(doc.FileID.UUID[0:16]),
			Page:   int32(doc.Page),
			Text:   doc.Content,
		})
	}

	return apiAnswer, nil
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
		a.logger.Sugar().With("error", err).Error("error listing screenings")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error listing screenings: %w", err))
		return
	}

	apiScreenings, err := mapScreenings(screenings)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error mapping screenings: %w", err))
		return
	}

	renderJSON(w, apiScreenings)
}

func mapScreenings(screenings []*ragserver.Screening) (api.Screenings, error) {
	apiScreenings := api.Screenings{
		Screenings: make([]api.Screening, 0, len(screenings)),
	}
	for _, aScreening := range screenings {
		apiScreening, err := mapScreening(aScreening)
		if err != nil {
			return api.Screenings{}, err
		}
		apiScreenings.Screenings = append(apiScreenings.Screenings, apiScreening)
	}
	return apiScreenings, nil
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
		a.logger.Sugar().With("error", err).Error("invalid screening ID")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid screening ID: %w", err))
		return
	}

	aScreening, err := a.ragServer.FindScreening(ctx, principal, ragserver.ScreeningID{UUID: screeningID})
	if err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("screening not found"))
			return
		}
		a.logger.Sugar().With("error", err).Error("error finding screening")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error finding screening: %w", err))
		return
	}

	apiScreening, err := mapScreening(aScreening)
	if err != nil {
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error mapping screening: %w", err))
		return
	}

	renderJSON(w, apiScreening)
}

// Delete a screening by ID
// (DELETE /screenings/{id})
func (a *Adapter) DeleteScreeningById(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var (
		ctx, cancel = context.WithTimeout(r.Context(), defaultTimeout)
		principal   = a.principalFromRequest(r)
	)
	defer cancel()

	screeningID, err := uuid.FromString(id.String())
	if err != nil {
		a.logger.Sugar().With("error", err).Error("invalid screening ID")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("invalid screening ID: %w", err))
		return
	}

	if err := a.ragServer.DeleteScreening(ctx, principal, ragserver.ScreeningID{UUID: screeningID}); err != nil {
		if errors.Is(err, ragserver.ErrNotFound) {
			renderJSONError(w, http.StatusNotFound, fmt.Errorf("screening not found"))
			return
		}
		a.logger.Sugar().With("error", err).Error("error finding screening")
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error finding screening: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
