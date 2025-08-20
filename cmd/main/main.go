package main

import (
	"cmp"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/neurosnap/sentences"
	"github.com/spf13/viper"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver/api"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/document"
	genaiAdapter "github.com/RichardKnop/ragserver/internal/ragserver/adapter/genai"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/pdf"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/rest"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/store"
	weaviateAdapter "github.com/RichardKnop/ragserver/internal/ragserver/adapter/weaviate"
	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

//go:embed testdata/english.json
var testEn string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("fatal error config file: ", err)
	}

	wvClient, err := weaviate.NewClient(weaviate.Config{
		Host:   "localhost:" + cmp.Or(os.Getenv("WVPORT"), "9035"),
		Scheme: "http",
	})
	if err != nil {
		log.Fatal("weaviate client: ", err)
	}
	wvAdapter, err := weaviateAdapter.New(ctx, wvClient)
	if err != nil {
		log.Fatal("weaviate adapter: ", err)
	}

	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal("genai client: ", err)
	}
	gAdapter := genaiAdapter.New(genaiClient)

	// Load the training data
	training, err := sentences.LoadTraining([]byte(testEn))
	if err != nil {
		log.Fatal("load training: ", err)
	}

	// Connect to the database
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rwc&cache=shared", viper.GetString("db.name")))
	if err != nil {
		log.Fatal("db open: ", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("db ping: ", err)
	}

	// Run db migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Fatal("migration driver: ", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+viper.GetString("db.migrations.path"),
		"sqlite3", driver)
	if err != nil {
		log.Fatal("migrations: ", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("migrations up: ", err)
	}

	var extractAdapter ragserver.ExtractAdapter
	switch viper.GetString("adapter.extract") {
	case "pdf":
		extractAdapter = pdf.New(training)
	case "document":
		extractAdapter = document.New(genaiClient, training)
	default:
		log.Fatalf("unknown extract adapter: %s", viper.GetString("extract.adapter"))
	}

	var (
		storeAdapter = store.New(db)
		rs           = ragserver.New(gAdapter, wvAdapter, training, extractAdapter, storeAdapter)
		restAdapter  = rest.New(rs)
		mux          = http.NewServeMux()
		// get an `http.Handler` that we can use
		h       = api.HandlerFromMux(restAdapter, mux)
		address = viper.GetString("server.host") + ":" + viper.GetString("server.port")
	)

	httpServer := &http.Server{
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Addr:              address,
		Handler:           h,
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
