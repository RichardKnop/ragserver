package document

import (
	"log"

	"github.com/neurosnap/sentences"
	"google.golang.org/genai"
)

type Adapter struct {
	client   *genai.Client
	training *sentences.Storage
	model    string
}

type Option func(*Adapter)

func WithModel(model string) Option {
	return func(a *Adapter) {
		a.model = model
	}
}

const defaultModel = "gemini-2.5-flash"

func New(client *genai.Client, training *sentences.Storage, options ...Option) *Adapter {
	a := &Adapter{
		client:   client,
		training: training,
		model:    defaultModel,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init google document adapter,",
		"model:", a.model,
	)

	return a
}
