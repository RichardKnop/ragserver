package pdf

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/neurosnap/sentences"
)

type Adapter struct {
	httpClient *http.Client
	baseURL    string
	training   *sentences.Storage
	logger     *zap.Logger
}

type Option func(*Adapter)

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

func WithBaseURL(url string) Option {
	return func(a *Adapter) {
		a.baseURL = url
	}
}

func WithHttpClient(client *http.Client) Option {
	return func(a *Adapter) {
		a.httpClient = client
	}
}

func New(training *sentences.Storage, options ...Option) *Adapter {
	a := &Adapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "http://pdf-document-layout-analysis:5060",
		training:   training,
		logger:     zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	a.logger.Sugar().With(
		"base URL", a.baseURL,
	).Info("init pdf adapter")

	return a
}
