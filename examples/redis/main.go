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
	_ "github.com/mattn/go-sqlite3"
	"github.com/neurosnap/sentences"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"google.golang.org/genai"

	"github.com/RichardKnop/ragserver"
	"github.com/RichardKnop/ragserver/adapter/document"
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
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("fatal error config file: ", err)
	}

	// The client gets the API key from the environment variable `GEMINI_API_KEY`.
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal("genai client: ", err)
	}

	// Load the training data
	training, err := sentences.LoadTraining([]byte(ragserver.TestEn))
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
	if err := ragserver.Migrate(db); err != nil {
		log.Fatal("db migrate: ", err)
	}

	var embebber ragserver.Embedder
	switch viper.GetString("adapter.embed.name") {
	case "google-genai":
		log.Println("embed adapter: google-genai")
		embebber = googlegenai.New(
			genaiClient,
			googlegenai.WithEmbeddingModel(viper.GetString("adapter.embed.model")),
		)
	case "hugot":
		log.Println("embed adapter: hugot")
		session, err := hugot.NewGoSession()
		if err != nil {
			log.Fatal("hugot session: ", err)
		}
		defer func() {
			err := session.Destroy()
			if err != nil {
				log.Fatal("hugot session destroy: ", err)
			}
		}()
		embebber, err = hugotAdapter.New(
			session,
			hugotAdapter.WithModel(viper.GetString("adapter.embed.model")),
		)
		if err != nil {
			log.Fatal("hugot adapter: ", err)
		}
	default:
		log.Fatalf("unknown embed adapter: %s", viper.GetString("adapter.embed"))
	}

	var retriever ragserver.Retriever
	switch viper.GetString("adapter.retrieve.name") {
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
	switch viper.GetString("adapter.extract.name") {
	case "pdf":
		log.Println("extract adapter: pdf")
		extractor = pdf.New(training)
	case "document":
		log.Println("extract adapter: document")
		extractor = document.New(
			genaiClient,
			training,
			document.WithModel(viper.GetString("adapter.extract.model")),
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

	lm := googlegenai.New(
		genaiClient,
		googlegenai.WithGenerativeModel(viper.GetString("adapter.generative.model")),
	)

	var (
		storeAdapter = store.New(db)
		rs           = ragserver.New(extractor, embebber, retriever, lm, storeAdapter, opts...)
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
