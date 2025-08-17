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
	cdr := new(CreateDocumentsRequest)

	if err := readRequestJSON(req, cdr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()

	if err := a.ragServer.CreateDocuments(ctx, cdr.Documents); err != nil {
		log.Printf("error creating documents: %v", err)
		http.Error(w, "error creating documents", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
