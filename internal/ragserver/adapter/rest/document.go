package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RichardKnop/ragserver/api"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

// Add documents to the knowledge base
// (POST /documents)
func (a *Adapter) AddDocuments(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		principal = a.principalFromRequest(r)
	)

	apiRequest := new(api.Documents)

	if err := readRequestJSON(r, apiRequest); err != nil {
		renderJSONError(w, http.StatusBadGateway, err)
		return
	}

	documents := make([]ragserver.Document, len(apiRequest.Documents))
	for i, doc := range apiRequest.Documents {
		documents[i] = ragserver.Document{
			Text: doc.Text,
		}
	}

	if err := a.ragServer.CreateDocuments(ctx, principal, documents); err != nil {
		log.Printf("error creating documents: %v", err)
		renderJSONError(w, http.StatusInternalServerError, fmt.Errorf("error creating documents: %w", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
