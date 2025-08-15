package main

import (
	"net/http"

	"github.com/neurosnap/sentences"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"google.golang.org/genai"
)

type ragServer struct {
	wvClient *weaviate.Client
	client   *genai.Client
	training *sentences.Storage
}

func (rs *ragServer) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /files/", rs.uploadFileHandler)
	mux.HandleFunc("POST /add/", rs.addDocumentsHandler)
	mux.HandleFunc("POST /query/", rs.queryHandler)
}
