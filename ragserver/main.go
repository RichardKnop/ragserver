package main

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/neurosnap/sentences"
	"google.golang.org/genai"
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

	pdf.DebugOn = true

	ragServer := &ragServer{
		wvClient: wvClient,
		client:   genaiClient,
		training: training,
	}

	mux := http.NewServeMux()
	ragServer.registerHandlers(mux)

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
