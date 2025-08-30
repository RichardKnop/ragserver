package hugot

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelineBackends"
	"github.com/knights-analytics/hugot/pipelines"
)

type Adapter struct {
	session         *hugot.Session
	embedding       *pipelines.FeatureExtractionPipeline
	generative      *pipelines.TextGenerationPipeline
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

func New(ctx context.Context, session *hugot.Session, options ...Option) (*Adapter, error) {
	a := &Adapter{
		session: session,
	}

	for _, o := range options {
		o(a)
	}

	log.Println(
		"init hugot adapter,",
		"embedding model:", a.embeddingModel,
		"generative model:", a.generativeModel,
	)

	return a, a.init(ctx)
}

const adapterName = "hugot"

func (a *Adapter) Name() string {
	return adapterName
}

func (a *Adapter) init(ctx context.Context) error {
	if a.embeddingModel == "" && a.generativeModel == "" {
		return fmt.Errorf("either embedding model or generative model must be specified")
	}

	var (
		wg            = new(sync.WaitGroup)
		embeddingErr  error
		generativeErr error
	)

	if a.embeddingModel != "" {
		wg.Go(func() {
			log.Println("downloading embedding model:", a.embeddingModel)

			downloadOptions := hugot.NewDownloadOptions()
			downloadOptions.OnnxFilePath = "onnx/model.onnx" // Specify which ONNX file to use
			modelPath, err := hugot.DownloadModel(a.embeddingModel, "./models/", downloadOptions)
			if err != nil {
				embeddingErr = fmt.Errorf("failed to download embedding model: %w", err)
				return
			}

			// Create feature extraction pipeline configuration
			config := hugot.FeatureExtractionConfig{
				ModelPath: modelPath,
				Name:      "embeddingPipeline",
			}

			// Create the feature extraction pipeline
			a.embedding, err = hugot.NewPipeline(a.session, config)
			if err != nil {
				embeddingErr = fmt.Errorf("failed to create embedding pipeline: %w", err)
				return
			}
		})
	}

	if a.generativeModel != "" {
		wg.Go(func() {
			log.Println("downloading generative model:", a.generativeModel)

			modelPath, err := hugot.DownloadModel(a.generativeModel, "./models/", hugot.NewDownloadOptions())
			if err != nil {
				generativeErr = fmt.Errorf("failed to download generative model: %w", err)
				return
			}

			// Create text generation pipeline configuration
			config := hugot.TextGenerationConfig{
				ModelPath:    modelPath,
				Name:         "textGenerationPipeline",
				OnnxFilename: "onnx/model.onnx",
				Options: []pipelineBackends.PipelineOption[*pipelines.TextGenerationPipeline]{
					pipelines.WithMaxTokens(200),
				},
			}

			// Create the text extraction pipeline
			a.generative, err = hugot.NewPipeline(a.session, config)
			if err != nil {
				generativeErr = fmt.Errorf("failed to create generative pipeline: %w", err)
				return
			}
		})
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		done <- struct{}{}
		close(done)
	}()

	select {
	case <-ctx.Done(): // context cancelled or timed out
		return ctx.Err()
	case <-done: // wait group finished
		if embeddingErr != nil {
			return fmt.Errorf("failed to create embedding pipeline: %w", embeddingErr)
		}
		if generativeErr != nil {
			return fmt.Errorf("failed to create generative pipeline: %w", generativeErr)
		}

		return nil
	}
}
