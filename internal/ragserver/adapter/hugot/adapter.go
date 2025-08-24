package hugot

import (
	"log"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

type Adapter struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	model    string
}

type Option func(*Adapter)

const defaultModel = "all-MiniLM-L6-v2"

func WithModel(model string) Option {
	return func(a *Adapter) {
		a.model = model
	}
}

func New(session *hugot.Session, options ...Option) (*Adapter, error) {
	a := &Adapter{
		session: session,
		model:   defaultModel,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init hugot adapter,",
		"model:", a.model,
	)

	return a, a.init()
}

const adapterName = "hugot"

func (a *Adapter) Name() string {
	return adapterName
}

func (a *Adapter) init() error {
	// Download the model
	downloadOptions := hugot.NewDownloadOptions()
	downloadOptions.OnnxFilePath = "onnx/model.onnx" // Specify which ONNX file to use
	modelPath, err := hugot.DownloadModel("sentence-transformers/"+a.model, "./models/", downloadOptions)
	if err != nil {
		return err
	}

	// Create feature extraction pipeline configuration
	config := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "embeddingPipeline",
	}

	// Create the feature extraction pipeline
	a.pipeline, err = hugot.NewPipeline(a.session, config)
	if err != nil {
		return nil
	}

	return err
}
