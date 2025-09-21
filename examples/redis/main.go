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

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/knights-analytics/hugot"
	hugotOptions "github.com/knights-analytics/hugot/options"
	_ "github.com/lib/pq"
	"github.com/neurosnap/sentences"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/adapter/document"
	"github.com/RichardKnop/ragserver/adapter/filestorage"
	googlegenai "github.com/RichardKnop/ragserver/adapter/google-genai"
	hugotAdapter "github.com/RichardKnop/ragserver/adapter/hugot"
	"github.com/RichardKnop/ragserver/adapter/pdf"
	redisAdapter "github.com/RichardKnop/ragserver/adapter/redis"
	"github.com/RichardKnop/ragserver/adapter/rest"
	"github.com/RichardKnop/ragserver/adapter/store"
	"github.com/RichardKnop/ragserver/api"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("examples/redis")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("fatal error config file: ", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("logger: ", err)
	}
	defer logger.Sync() // flushes buffer, if any

	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := initGenaiClient(ctx)
	if err != nil {
		log.Fatal("genai client: ", err)
	}

	// Load the training data
	training, err := sentences.LoadTraining([]byte(ragserver.TestEn))
	if err != nil {
		log.Fatal("load training: ", err)
	}

	// Connect to the database
	log.Println("connecting to db: ", viper.GetString("db.name"))
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			viper.GetString("db.user"),
			viper.GetString("db.password"),
			viper.GetString("db.host"),
			viper.GetString("db.port"),
			viper.GetString("db.name"),
			viper.GetString("db.sslmode"),
		),
	)
	if err != nil {
		log.Fatal("db open:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}

	// Run db migrations
	if err := ragserver.Migrate(db); err != nil {
		log.Fatal("db migrate:", err)
	}

	// Extractor
	var extractor ragserver.Extractor
	switch name := viper.GetString("adapter.extract.name"); name {
	case "pdf":
		log.Println("extract adapter: pdf")
		extractor = pdf.New(training, pdf.WithLogger(logger))
	case "document":
		log.Println("extract adapter: document")
		extractor = document.New(
			genaiClient,
			training,
			document.WithModel(viper.GetString("adapter.extract.model")),
		)
	default:
		log.Fatalf("unknown extract adapter: %s", name)
	}

	//Hugot session
	hAdapter, hugotCancel, err := initHugot(ctx, logger)
	if err != nil {
		log.Fatal("hugot init: ", err)
	}
	if hugotCancel != nil {
		defer hugotCancel()
	}

	// Embedder
	var embebber ragserver.Embedder
	switch name := viper.GetString("adapter.embed.name"); name {
	case "hugot":
		log.Println("embed adapter: hugot")
		embebber = hAdapter
	case "google-genai":
		log.Println("embed adapter: google-genai")
		embebber = googlegenai.New(
			genaiClient,
			googlegenai.WithEmbeddingModel(viper.GetString("adapter.embed.model")),
			googlegenai.WithLogger(logger),
		)
	default:
		log.Fatalf("unknown embed adapter: %s", name)
	}

	// Retriever
	var retriever ragserver.Retriever
	switch name := viper.GetString("adapter.retrieve.name"); name {
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
			redisAdapter.WithLogger(logger),
		)
		if err != nil {
			log.Fatal("redis adapter: ", err)
		}
	default:
		log.Fatalf("unknown retrieve adapter: %s", name)
	}

	// Generative model
	var gm ragserver.GenerativeModel
	switch name := viper.GetString("adapter.generative.name"); name {
	case "hugot":
		log.Println("generative adapter: hugot")
		gm = hAdapter
	case "google-genai":
		log.Println("generative adapter: google-genai")
		gm = googlegenai.New(
			genaiClient,
			googlegenai.WithGenerativeModel(viper.GetString("adapter.generative.model")),
			googlegenai.WithTemplatesDir(viper.GetString("adapter.generative.templates_dir")),
			googlegenai.WithLogger(logger),
		)
	default:
		log.Fatalf("unknown generative adapter: %s", name)
	}

	// Relevant topics to limit context
	relevantTopics, err := relevantTopicsFromConfig()
	if err != nil {
		log.Fatal("relevant topics: ", err)
	}
	log.Println("relevant topics configured", relevantTopics)

	opts := []ragserver.Option{
		ragserver.WithRelevantTopics(relevantTopics),
		ragserver.WithLogger(logger),
	}

	fileStorage, err := filestorage.New(
		filestorage.WithDir(viper.GetString("adapter.filestorage.dir")),
		filestorage.WithLogger(logger),
	)
	if err != nil {
		log.Fatal("file storage: ", err)
	}

	var (
		storeAdapter = store.New(db)
		rs           = ragserver.New(
			extractor,
			embebber,
			retriever,
			gm,
			storeAdapter,
			fileStorage,
			opts...,
		)
		restAdapter = rest.New(rs, rest.WithLogger(logger))
		mux         = http.NewServeMux()
		// get an `http.Handler` that we can use
		h       = api.HandlerFromMux(restAdapter, mux)
		address = ":" + viper.GetString("http.port")
	)

	httpServer := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       10 * time.Second,
		Addr:              address,
		Handler:           api.RecoveryMiddleware(h),
	}

	log.Println("listening on", address)

	go func() {
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	stopProcessingFiles := rs.ProcessFiles(ctx)
	defer stopProcessingFiles()

	stopProcessingScreenings := rs.ProcessScreenings(ctx)
	defer stopProcessingScreenings()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	cancel()

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

func initGenaiClient(ctx context.Context) (*genai.Client, error) {
	if viper.GetString("adapter.extract.name") != "document" &&
		viper.GetString("adapter.embed.name") != "google-genai" &&
		viper.GetString("adapter.generative.name") != "google-genai" {
		return nil, nil
	}
	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("genai client: %w", err)
	}

	return genaiClient, nil
}

func initHugot(ctx context.Context, logger *zap.Logger) (*hugotAdapter.Adapter, func(), error) {
	if viper.GetString("hugot.backend") == "" {
		return nil, nil, nil
	}
	// Hugot session
	var session *hugot.Session
	switch backend := viper.GetString("hugot.backend"); backend {
	case "go":
		log.Println("hugot backend: go")
		var err error
		session, err = hugot.NewGoSession()
		if err != nil {
			return nil, nil, fmt.Errorf("hugot session: %w", err)
		}
	case "ort":
		log.Println("hugot backend: ort")

		// Check if onnxruntime was installed
		onnxPath := viper.GetString("hugot.onnxruntime_path")
		if _, err := os.Stat(onnxPath); errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("onnxruntime backend selected but %s does not exist", onnxPath)
		}
		var err error
		session, err = hugot.NewORTSession(
			hugotOptions.WithOnnxLibraryPath(onnxPath),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("hugot session: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unknown hugot backend: %s", backend)
	}
	cancel := func() {
		err := session.Destroy()
		if err != nil {
			log.Fatal("hugot session destroy: ", err)
		}
	}
	hAdapter, err := hugotAdapter.New(
		ctx,
		session,
		hugotAdapter.WithEmbeddingModelName(viper.GetString("adapter.embed.model")),
		hugotAdapter.WithEmbeddingModelOnnxFilePath(viper.GetString("adapter.embed.onx_file_path")),
		hugotAdapter.WithGenerativeModelName(viper.GetString("adapter.generative.model")),
		hugotAdapter.WithGenerativeModelOnnxFilePath(viper.GetString("adapter.generative.onx_file_path")),
		hugotAdapter.WithGenerativeModelExternalDataPath(viper.GetString("adapter.generative.external_data_path")),
		hugotAdapter.WithTemplatesDir(viper.GetString("adapter.generative.templates_dir")),
		hugotAdapter.WithModelsDir(viper.GetString("hugot.models_dir")),
		hugotAdapter.WithLogger(logger),
	)
	if err != nil {
		log.Fatal("hugot adapter: ", err)
	}
	return hAdapter, cancel, nil
}
