package googlegenai

import (
	"log"

	"google.golang.org/genai"
)

type Adapter struct {
	client          *genai.Client
	embeddingModel  string
	generativeModel string
}

type Option func(*Adapter)

func WithEmbeddingModel(model string) Option {
	return func(a *Adapter) {
		a.embeddingModel = model
	}
}

func WithGenerativeModel(model string) Option {
	return func(a *Adapter) {
		a.generativeModel = model
	}
}

func New(client *genai.Client, options ...Option) *Adapter {
	a := &Adapter{
		client: client,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init google genai adapter,",
		"embedding model:", a.embeddingModel,
		"generative model:", a.generativeModel,
	)

	return a
}

const adapterName = "google-genai"

func (a *Adapter) Name() string {
	return adapterName
}
