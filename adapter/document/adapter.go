package document

import (
	"github.com/neurosnap/sentences"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

type Adapter struct {
	client   *genai.Client
	training *sentences.Storage
	model    string
	logger   *zap.Logger
}

type Option func(*Adapter)

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

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
		logger:   zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	a.logger.Sugar().With("model", a.model).Info("init google document adapter")

	return a
}
