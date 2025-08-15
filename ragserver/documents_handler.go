package main

import (
	"log"
	"net/http"

	"github.com/weaviate/weaviate/entities/models"
	"google.golang.org/genai"
)

type Document struct {
	Text string `json:"text"`
}
type AddRequest struct {
	Documents []Document `json:"documents"`
}

func (rs *ragServer) addDocumentsHandler(w http.ResponseWriter, req *http.Request) {
	// Parse HTTP request from JSON.
	ar := new(AddRequest)

	err := readRequestJSON(req, ar)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()

	// Use the batch embedding API to embed all documents at once.
	//  result, err := client.Models.Embeddings.
	contents := make([]*genai.Content, 0, len(ar.Documents))
	for _, doc := range ar.Documents {
		contents = append(contents, genai.NewContentFromText(doc.Text, genai.RoleUser))
	}
	embedResponse, err := rs.client.Models.EmbedContent(ctx,
		embeddingModelName,
		contents,
		nil,
	)
	log.Printf("invoking embedding model with %v documents", len(ar.Documents))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(embedResponse.Embeddings) != len(ar.Documents) {
		http.Error(w, "embedded batch size mismatch", http.StatusInternalServerError)
		return
	}

	// Convert our documents - along with their embedding vectors - into types
	// used by the Weaviate client library.
	objects := make([]*models.Object, len(ar.Documents))
	for i, doc := range ar.Documents {
		objects[i] = &models.Object{
			Class: "Document",
			Properties: map[string]any{
				"text": doc.Text,
			},
			Vector: embedResponse.Embeddings[i].Values,
		}
	}

	// Store documents with embeddings in the Weaviate DB.
	log.Printf("storing %v objects in weaviate", len(objects))
	_, err = rs.wvClient.Batch().ObjectsBatcher().WithObjects(objects...).Do(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
