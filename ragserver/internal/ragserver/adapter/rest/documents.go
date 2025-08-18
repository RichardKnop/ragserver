package rest

import (
	"log"
	"net/http"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

type CreateDocumentsRequest struct {
	Documents []ragserver.Document `json:"documents"`
}

func (a *Adapter) createDocumentsHandler(w http.ResponseWriter, req *http.Request) {
	var (
		ctx       = req.Context()
		principal = a.principalFromRequest(req)
	)

	cdr := new(CreateDocumentsRequest)

	if err := readRequestJSON(req, cdr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.ragServer.CreateDocuments(ctx, principal, cdr.Documents); err != nil {
		log.Printf("error creating documents: %v", err)
		http.Error(w, "error creating documents", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
