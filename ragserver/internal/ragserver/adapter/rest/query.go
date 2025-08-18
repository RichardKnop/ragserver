package rest

import (
	"log"
	"net/http"

	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type QueryRequest struct {
	Content string   `json:"content"`
	Type    string   `json:"type"`
	FileIDs []string `json:"file_ids"`
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

	var (
		query = ragserver.Query{
			Type: ragserver.QueryType(qr.Type),
			Text: qr.Content,
		}
		fileIDs []ragserver.FileID
	)

	for _, id := range qr.FileIDs {
		fileID, err := uuid.FromString(id)
		if err != nil {
			log.Printf("invalid file ID: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileIDs = append(fileIDs, ragserver.FileID{fileID})
	}

	if len(fileIDs) == 0 {
		http.Error(w, "missing file_ids", http.StatusBadRequest)
		return
	}

	responses, err := a.ragServer.Generate(ctx, principal, query, fileIDs...)
	if err != nil {
		log.Printf("error generating response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderJSON(w, QueryResponse{
		Responses: responses,
	})
}
