package main

import (
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
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver/api"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/document"
	googlegenai "github.com/RichardKnop/ragserver/internal/ragserver/adapter/google-genai"
	"github.com/RichardKnop/ragserver/internal/ragserver/adapter/pdf"
	redisAdapter "github.com/RichardKnop/ragserver/internal/ragserver/adapter/redis"
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

	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal("genai client: ", err)
	}
	genaiAdapter := googlegenai.New(
		genaiClient,
		googlegenai.WithEmbeddingModel(viper.GetString("gemini.models.embeddings")),
		googlegenai.WithGenerativeModel(viper.GetString("gemini.models.generative")),
	)

	// Load the training data
	training, err := sentences.LoadTraining([]byte(testEn))
	if err != nil {
		log.Fatal("load training: ", err)
	}

	// Connect to the database
	log.Println("connecting to db:", viper.GetString("db.name"))
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

	var embebber ragserver.Embedder
	switch viper.GetString("adapter.embed") {
	case "google-genai":
		log.Println("embed adapter: google-genai")
		embebber = genaiAdapter
	default:
		log.Fatalf("unknown embed adapter: %s", viper.GetString("adapter.embed"))
	}

	var retriever ragserver.Retriever
	switch viper.GetString("adapter.retrieve") {
	case "weaviate":
		log.Println("retrieve adapter: weaviate")
		wvClient, err := weaviate.NewClient(weaviate.Config{
			Host:   viper.GetString("weaviate.addr"),
			Scheme: "http",
		})
		if err != nil {
			log.Fatal("weaviate client: ", err)
		}
		retriever, err = weaviateAdapter.New(ctx, wvClient)
		if err != nil {
			log.Fatal("weaviate adapter: ", err)
		}
	case "redis":
		log.Println("retrieve adapter: redis")
		rdb := redis.NewClient(&redis.Options{
			Addr:     viper.GetString("redis.addr"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			Protocol: viper.GetInt("redis.protocol"),
		})
		var err error
		retriever, err = redisAdapter.New(
			ctx,
			rdb,
			redisAdapter.WithIndexName(viper.GetString("redis.index")),
			redisAdapter.WithIndexPrefix(viper.GetString("redis.index_prefix")),
			redisAdapter.WithDialectVersion(viper.GetInt("redis.protocol")),
			redisAdapter.WithVectorDim(viper.GetInt("redis.vector_dim")),
			redisAdapter.WithVectorDistanceMetric(viper.GetString("redis.vector_distance_metric")),
		)
		if err != nil {
			log.Fatal("redis adapter: ", err)
		}
	default:
		log.Fatalf("unknown retrieve adapter: %s", viper.GetString("adapter.retrieve"))
	}

	var extractor ragserver.Extractor
	switch viper.GetString("adapter.extract") {
	case "pdf":
		log.Println("extract adapter: pdf")
		extractor = pdf.New(training)
	case "document":
		log.Println("extract adapter: document")
		extractor = document.New(
			genaiClient,
			training,
			document.WithGenerativeModel(viper.GetString("gemini.models.generative")),
		)
	default:
		log.Fatalf("unknown extract adapter: %s", viper.GetString("adapter.extract"))
	}

	relevantTopics, err := relevantTopicsFromConfig()
	if err != nil {
		log.Fatal("relevant topics: ", err)
	}
	log.Println("relevant topics configured", relevantTopics)
	opts := []ragserver.Option{
		ragserver.WithRelevantTopics(relevantTopics),
	}

	var (
		storeAdapter = store.New(db)
		rs           = ragserver.New(extractor, embebber, retriever, genaiAdapter, storeAdapter, opts...)
		restAdapter  = rest.New(rs)
		mux          = http.NewServeMux()
		// get an `http.Handler` that we can use
		h       = api.HandlerFromMux(restAdapter, mux)
		address = viper.GetString("http.host") + ":" + viper.GetString("http.port")
	)

	httpServer := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       10 * time.Second,
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

func relevantTopicsFromConfig() (ragserver.RelevantTopics, error) {
	var relevantTopics []ragserver.Topic
	for name, keywords := range viper.GetStringMapStringSlice("relevant_topics") {
		relevantTopics = append(relevantTopics, ragserver.Topic{
			Name:     name,
			Keywords: keywords,
		})
	}
	return relevantTopics, nil
}
