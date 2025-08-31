package hugot

import (
	"context"
	"fmt"
	"log"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelineBackends"
	"github.com/knights-analytics/hugot/pipelines"
)

type modelConfig struct {
	name             string
	externalDataPath string
}

type Adapter struct {
	session          *hugot.Session
	embedding        *pipelines.FeatureExtractionPipeline
	generative       *pipelines.TextGenerationPipeline
	embeddingConfig  modelConfig
	generativeConfig modelConfig
	modelsDir        string
}

type Option func(*Adapter)

func WithEmbeddingModelName(name string) Option {
	return func(a *Adapter) {
		a.embeddingConfig.name = name
	}
}

func WithGenerativeModelName(name string) Option {
	return func(a *Adapter) {
		a.generativeConfig.name = name
	}
}

func WithGenerativeModelExternalDataPath(path string) Option {
	return func(a *Adapter) {
		a.generativeConfig.externalDataPath = path
	}
}

func WithModelsDir(path string) Option {
	return func(a *Adapter) {
		a.modelsDir = path
	}
}

const defaultModelsDir = "/models"

func New(ctx context.Context, session *hugot.Session, options ...Option) (*Adapter, error) {
	a := &Adapter{
		session:          session,
		embeddingConfig:  modelConfig{},
		generativeConfig: modelConfig{},
		modelsDir:        defaultModelsDir,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init hugot adapter,",
		"embedding model config:", a.embeddingConfig,
		"generative model config:", a.generativeConfig,
		"models dir:", a.modelsDir,
	)

	if err := a.init(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

const adapterName = "hugot"

func (a *Adapter) Name() string {
	return adapterName
}

func (a *Adapter) init(ctx context.Context) error {
	if a.embeddingConfig.name == "" && a.generativeConfig.name == "" {
		return fmt.Errorf("either embedding model or generative model must be specified")
	}

	if a.embeddingConfig.name != "" {
		log.Println("start downloading embedding model:", a.embeddingConfig.name)

		downloadOptions := hugot.NewDownloadOptions()
		downloadOptions.OnnxFilePath = "onnx/model.onnx"
		modelPath, err := hugot.DownloadModel(a.embeddingConfig.name, a.modelsDir, downloadOptions)
		if err != nil {
			return fmt.Errorf("failed to download embedding model: %w", err)
		}

		// Create feature extraction pipeline configuration
		config := hugot.FeatureExtractionConfig{
			ModelPath: modelPath,
			Name:      "embeddingPipeline",
		}

		// Create the feature extraction pipeline
		a.embedding, err = hugot.NewPipeline(a.session, config)
		if err != nil {
			return fmt.Errorf("failed to create embedding pipeline: %w", err)
		}

		log.Println("downloaded embedding model:", a.embeddingConfig.name)
	}

	if a.generativeConfig.name != "" {
		log.Println("start downloading generative model:", a.generativeConfig.name)

		downloadOptions := hugot.NewDownloadOptions()
		downloadOptions.OnnxFilePath = "onnx/model.onnx"
		if a.generativeConfig.externalDataPath != "" {
			downloadOptions.ExternalDataPath = a.generativeConfig.externalDataPath
		}
		modelPath, err := hugot.DownloadModel(a.generativeConfig.name, a.modelsDir, downloadOptions)
		if err != nil {
			return fmt.Errorf("failed to download generative model: %w", err)
		}

		// Create text generation pipeline configuration
		config := hugot.TextGenerationConfig{
			ModelPath:    modelPath,
			Name:         "textGenerationPipeline",
			OnnxFilename: "onnx/model.onnx",
			Options: []pipelineBackends.PipelineOption[*pipelines.TextGenerationPipeline]{
				pipelines.WithMaxTokens(200),
				pipelines.WithGemmaTemplate(),
			},
		}

		// Create the text extraction pipeline
		a.generative, err = hugot.NewPipeline(a.session, config)
		if err != nil {
			return fmt.Errorf("failed to create generative pipeline: %w", err)
		}

		log.Println("downloaded generative model:", a.generativeConfig.name)
	}

	return nil
}
