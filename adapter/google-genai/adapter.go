package googlegenai

import (
	"go.uber.org/zap"
	"google.golang.org/genai"
)

type Adapter struct {
	client          *genai.Client
	embeddingModel  string
	generativeModel string
	templatesDir    string
	logger          *zap.Logger
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

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

const (
	defaultTemplatesDir = "templates/google-genai/"
)

func New(client *genai.Client, options ...Option) *Adapter {
	a := &Adapter{
		client:       client,
		templatesDir: defaultTemplatesDir,
		logger:       zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	a.logger.Sugar().With(
		"embedding model", a.embeddingModel,
		"generative model", a.generativeModel,
		"templates dir", a.templatesDir,
	).Info("init google genai adapter")

	return a
}

const adapterName = "google-genai"

func (a *Adapter) Name() string {
	return adapterName
}
