package main

import (
	"cmp"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/genai"
)

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

	ragServer := &ragServer{
		ctx:      ctx,
		wvClient: wvClient,
		client:   genaiClient,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /add/", ragServer.addDocumentsHandler)
	mux.HandleFunc("POST /query/", ragServer.queryHandler)

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
