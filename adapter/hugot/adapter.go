package hugot

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelineBackends"
	"github.com/knights-analytics/hugot/pipelines"
	"go.uber.org/zap"
)

type modelConfig struct {
	name             string
	onxFilePath      string
	externalDataPath string
}

type Adapter struct {
	session          *hugot.Session
	embedding        *pipelines.FeatureExtractionPipeline
	generative       *pipelines.TextGenerationPipeline
	embeddingConfig  modelConfig
	generativeConfig modelConfig
	templatesDir     string
	modelsDir        string
	logger           *zap.Logger
}

type Option func(*Adapter)

func WithLogger(logger *zap.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

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

func WithEmbeddingModelOnnxFilePath(path string) Option {
	return func(a *Adapter) {
		a.embeddingConfig.onxFilePath = path
	}
}

func WithGenerativeModelOnnxFilePath(path string) Option {
	return func(a *Adapter) {
		a.generativeConfig.onxFilePath = path
	}
}

func WithGenerativeModelExternalDataPath(path string) Option {
	return func(a *Adapter) {
		a.generativeConfig.externalDataPath = path
	}
}

func WithTemplatesDir(dir string) Option {
	return func(a *Adapter) {
		a.templatesDir = dir
	}
}

func WithModelsDir(path string) Option {
	return func(a *Adapter) {
		a.modelsDir = path
	}
}

const (
	defaultTemplatesDir = "templates/hugot/"
	defaultModelsDir    = "/models"
	defaultOnxFilePath  = "onnx/model.onnx"
)

func New(ctx context.Context, session *hugot.Session, options ...Option) (*Adapter, error) {
	a := &Adapter{
		session:          session,
		embeddingConfig:  modelConfig{onxFilePath: defaultOnxFilePath},
		generativeConfig: modelConfig{onxFilePath: defaultOnxFilePath},
		templatesDir:     defaultTemplatesDir,
		modelsDir:        defaultModelsDir,
		logger:           zap.NewNop(),
	}

	for _, o := range options {
		o(a)
	}

	a.logger.Sugar().With(
		"embedding model config", a.embeddingConfig,
		"generative model config", a.generativeConfig,
		"templates dir", a.templatesDir,
		"models dir", a.modelsDir,
	).Info("init hugot adapter")

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
		modelPath, err := checkModelExists(a.modelsDir, a.embeddingConfig.name)
		if err != nil {
			return fmt.Errorf("failed to check embedding model: %w", err)
		}

		if modelPath == "" {
			a.logger.Sugar().Info("start downloading embedding model:", a.embeddingConfig.name)

			downloadOptions := hugot.NewDownloadOptions()
			downloadOptions.OnnxFilePath = a.embeddingConfig.onxFilePath
			modelPath, err = hugot.DownloadModel(a.embeddingConfig.name, a.modelsDir, downloadOptions)
			if err != nil {
				return fmt.Errorf("failed to download embedding model: %w", err)
			}

			a.logger.Sugar().Info("downloaded embedding model:", a.embeddingConfig.name)
		} else {
			a.logger.Sugar().Info("embedding model already exists, skipping download:", modelPath)
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
	}

	if a.generativeConfig.name != "" {
		modelPath, err := checkModelExists(a.modelsDir, a.generativeConfig.name)
		if err != nil {
			return fmt.Errorf("failed to check generative model: %w", err)
		}

		if modelPath == "" {
			a.logger.Sugar().Info("start downloading generative model:", a.generativeConfig.name)

			downloadOptions := hugot.NewDownloadOptions()
			downloadOptions.OnnxFilePath = a.generativeConfig.onxFilePath
			if a.generativeConfig.externalDataPath != "" {
				downloadOptions.ExternalDataPath = a.generativeConfig.externalDataPath
			}
			modelPath, err = hugot.DownloadModel(a.generativeConfig.name, a.modelsDir, downloadOptions)
			if err != nil {
				return fmt.Errorf("failed to download generative model: %w", err)
			}

			a.logger.Sugar().Info("downloaded generative model:", a.generativeConfig.name)
		} else {
			a.logger.Sugar().Info("generative model already exists, skipping download:", modelPath)
		}

		// Create text generation pipeline configuration
		config := hugot.TextGenerationConfig{
			ModelPath:    modelPath,
			Name:         "textGenerationPipeline",
			OnnxFilename: a.generativeConfig.onxFilePath,
			Options: []pipelineBackends.PipelineOption[*pipelines.TextGenerationPipeline]{
				pipelines.WithMaxTokens(2096),
				pipelines.WithGemmaTemplate(),
			},
		}

		// Create the text extraction pipeline
		a.generative, err = hugot.NewPipeline(a.session, config)
		if err != nil {
			return fmt.Errorf("failed to create generative pipeline: %w", err)
		}
	}

	return nil
}

func checkModelExists(destination, modelName string) (string, error) {
	modelP := modelName
	if strings.Contains(modelP, ":") {
		modelP = strings.Split(modelName, ":")[0]
	}
	modelPath := path.Join(destination, strings.ReplaceAll(modelP, "/", "_"))

	_, err := os.Stat(modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return modelPath, nil
}
