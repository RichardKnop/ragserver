package rest

import (
	"log"
	"net/http"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type QueryRequest struct {
	Content string              `json:"content"`
	Type    ragserver.QueryType `json:"type"`
}

type QueryResponse struct {
	Responses []ragserver.Response `json:"responses"`
}

func (a *Adapter) queryHandler(w http.ResponseWriter, req *http.Request) {
	qr := new(QueryRequest)
	if err := readRequestJSON(req, qr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responses, err := a.ragServer.Generate(req.Context(), ragserver.Query{
		Type: qr.Type,
		Text: qr.Content,
	})
	if err != nil {
		log.Printf("error generating response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, QueryResponse{
		Responses: responses,
	})
}
