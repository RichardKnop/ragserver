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
	var (
		ctx       = req.Context()
		principal = a.principalFromRequest(req)
	)

	qr := new(QueryRequest)
	if err := readRequestJSON(req, qr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responses, err := a.ragServer.Generate(ctx, principal, ragserver.Query{
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
