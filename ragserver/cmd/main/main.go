package main

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neurosnap/sentences"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
	"google.golang.org/genai"

	"github.com/RichardKnop/ai/ragserver/internal/ragserver/adapter/pdf"
	"github.com/RichardKnop/ai/ragserver/internal/ragserver/adapter/rest"
	"github.com/RichardKnop/ai/ragserver/internal/ragserver/core/ragserver"
)

//go:embed testdata/english.json
var testEn string

var (
	port    = cmp.Or(os.Getenv("SERVERPORT"), "9020")
	address = "localhost:" + port
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wvClient, err := initWeaviate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// load the training data
	training, err := sentences.LoadTraining([]byte(testEn))
	if err != nil {
		log.Fatal(err)
	}

	pdfAdapter := pdf.New(training)
	rs := ragserver.New(wvClient, genaiClient, training, pdfAdapter)

	mux := http.NewServeMux()
	restAdapter := rest.New(rs)
	restAdapter.RegisterHandlers(mux)

	httpServer := &http.Server{
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Addr:              address,
		Handler:           mux,
	}

	log.Println("listening on", address)

	go func() {
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")
}

// initWeaviate initializes a weaviate client for our application.
func initWeaviate(ctx context.Context) (*weaviate.Client, error) {
	client, err := weaviate.NewClient(weaviate.Config{
		Host:   "localhost:" + cmp.Or(os.Getenv("WVPORT"), "9035"),
		Scheme: "http",
	})
	if err != nil {
		return nil, fmt.Errorf("initializing weaviate: %w", err)
	}

	// Create a new class (collection) in weaviate if it doesn't exist yet.
	cls := &models.Class{
		Class:      "Document",
		Vectorizer: "none",
	}
	exists, err := client.Schema().ClassExistenceChecker().WithClassName(cls.Class).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("weaviate error: %w", err)
	}
	if !exists {
		err = client.Schema().ClassCreator().WithClass(cls).Do(ctx)
		if err != nil {
			return nil, fmt.Errorf("weaviate error: %w", err)
		}
	}

	return client, nil
}
