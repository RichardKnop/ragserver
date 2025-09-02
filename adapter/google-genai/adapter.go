package googlegenai

import (
	"log"

	"google.golang.org/genai"
)

type Adapter struct {
	client          *genai.Client
	embeddingModel  string
	generativeModel string
	templatesDir    string
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

func WithTemplatesDir(dir string) Option {
	return func(a *Adapter) {
		a.templatesDir = dir
	}
}

const (
	defaultTemplatesDir = "templates/google-genai/"
)

func New(client *genai.Client, options ...Option) *Adapter {
	a := &Adapter{
		client:       client,
		templatesDir: defaultTemplatesDir,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init google genai adapter,",
		"embedding model:", a.embeddingModel,
		"generative model:", a.generativeModel,
		"templates dir:", a.templatesDir,
	)

	return a
}

const adapterName = "google-genai"

func (a *Adapter) Name() string {
	return adapterName
}
