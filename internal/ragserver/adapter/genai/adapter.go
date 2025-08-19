package genai

import (
	"google.golang.org/genai"
)

const (
	generativeModelName = "gemini-2.5-flash"
	embeddingModelName  = "text-embedding-004"
)

type Adapter struct {
	client              *genai.Client
	embeddingModelName  string
	generativeModelName string
}

type Option func(*Adapter)

func New(client *genai.Client, options ...Option) *Adapter {
	a := &Adapter{
		client:              client,
		embeddingModelName:  embeddingModelName,
		generativeModelName: generativeModelName,
	}

	for _, o := range options {
		o(a)
	}

	return a
}
