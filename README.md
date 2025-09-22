# RAG Server

- [RAG Server](#rag-server)
  - [Components](#components)
    - [Extractor](#extractor)
    - [Embedder](#embedder)
    - [Retriever](#retriever)
    - [GenerativeModel](#generativemodel)
- [Examples](#examples)
- [Database](#database)
- [Configuration](#configuration)
- [API](#api)
- [Adding Documents To Knowledge Base](#adding-documents-to-knowledge-base)
- [Screening](#screening)
  - [Questions Types](#questions-types)
  - [Create Screening](#create-screening)
- [Testing](#testing)

This project is a generic [RAG](https://cloud.google.com/use-cases/retrieval-augmented-generation?hl=en) server that can be used to answer questions using a knowledge base (corpus) refined from uploaded PDF documents. In the examples, I use ESG data about scope 1 and scope 2 emissions because that is what I have been testing the server with but it is built to be completely generic and flexible.

When querying the server, you can specify a query type and provide files that should contain the answer. The server uses embedding model to get a vector representation of the question and retrieve documents from the knowledge base that are most similar to the question. It will then generate a structured JSON answer depending on a query type as well as list of evidences (files) and specific pages in PDFs referencing where the answer was extracted from.

Main components of the RAG server are:

-  **Extractor**
-  **Embedder**
-  **Retriever**
-  **GenerativeModel**

These are defined as interfaces. You can implement your own components that implement these interface or use one of the provided implementations from `adapter/` folder.

```go
// Extractor extracts documents from various contents, optionally limited by relevant topics.
type Extractor interface {
	Extract(ctx context.Context, fileName string, contents io.ReadSeeker, topics RelevantTopics) ([]Document, error)
}

// Embedder encodes document passages as vectors
type Embedder interface {
	Name() string
	EmbedDocuments(ctx context.Context, documents []Document) ([]Vector, error)
	EmbedContent(ctx context.Context, content string) (Vector, error)
}

// Retriever that runs a question through the embeddings model and returns any encoded documents near the embedded question.
type Retriever interface {
	Name() string
	SaveDocuments(ctx context.Context, documents []Document, vectors []Vector) error
	ListDocumentsByFileID(ctx context.Context, id FileID) ([]Document, error)
	SearchDocuments(ctx context.Context, filter DocumentFilter, limit int) ([]Document, error)
}

// GenerativeModel uses generative AI to generate responses based on a query and relevant documents.
type GenerativeModel interface {
	Generate(ctx context.Context, question Question, documents []Document) ([]Response, error)
}
```

Then just simply create a new instance of RAG server and pass in adapters as inputs. You can hook up the REST adapter to any HTTP server, I just is the one from standard library:

```go
rs := ragserver.New(
  extractor, 
  embebber, 
  retriever, 
  gm, 
  storeAdapter,
  fileStorage,
)
restAdapter := rest.New(rs, rest.WithLogger(logger))
mux := http.NewServeMux()
h := api.HandlerFromMux(restAdapter, mux)
httpServer := &http.Server{
	ReadHeaderTimeout: 10 * time.Second,
	IdleTimeout:       10 * time.Second,
	Addr:              "localhost:8080",
	Handler:           h,
}
httpServer.ListenAndServe()
```

## Components

### Extractor

You can either use `adapter/pdf` (which does not depend on any API, it will just try to extract sentences from PDFs locally in code) or `adapter/document` which uses Gemini document vision to extract sentences from PDFs

### Embedder

You can use either the `adapter/google-genai` or `adapter/hugot` or implement your own.

### Retriever

You can use either the `adapter/redis` or `adapter/weaviate` or implement your own.

### GenerativeModel

You can use either the `adapter/google-genai` or `adapter/hugot` or implement your own.

# Examples

You can look at `examples/` folder to see different types of adapters in use. I suggest you create your own command line entrypoint though as the `github.com/RichardKnop/ragserver/server` package used by the examples imports all of the adapters and you can slim down on dependencies by only using specific adapters you want.

You can choose between [weaviate](https://github.com/weaviate/weaviate) and [redis](https://redis.io/) as a vector database.

For a quick test drive, you can run one of the examples:

Redis retriever backend, local PDF extractor, configurable to either use Gemini text embedding and generative model or hugot open source models:

```sh
docker compose -f examples/redis/docker-compose.yml up -d
```

Weaviate retriever backend, local PDF extractor, Gemini text embedding and generative model:

```sh
docker compose -f examples/weaviate/docker-compose.yml up -d
```

# Database

This project uses Postgres database. It is used to store information about uploaded files including file size, hash, content type etc. UUIDs from the SQL database should be referenced in the weaviate database as a `file_id` property.

# Configuration

If you are using Gemini for either text embeddings or to generate answers, you need to set `GEMINI_API_KEY` environment variable t.

For everything else, you can use whatever configuration method you prefer. If you use one of the examples, they rely on [viper](https://github.com/spf13/viper) to read configuration from a YAML config file.

Read `config.example.yaml` for a list of possible configuration options. However, this depends on your usage of this library so treat example config files and docker compose files just as examples.

# API

See the [OpenAPI spec](/api/api.yaml) for API reference.

# Adding Documents To Knowledge Base

Upload PDF files which will be used to extract documents:

```sh
./scripts/upload-file.sh '/Users/richardknop/Desktop/Statement on Emissions.pdf'
./scripts/upload-file.sh '/Users/richardknop/Desktop/TCFD Report.pdf'
```

Keep track of file IDs because those are required to query the LLM for an answer.

You can list all current files:

```sh
./scripts/list-files.sh
```

To list documents extracted from a specific file (currently limited to 100 documents, no pagination support):

```sh
./scripts/list-file-documents.sh 9b3e8b3d-b62b-4434-920f-858f44429596
```

To perform similary search on file documents limited to top 25 documents:

```sh
./scripts/list-file-documents.sh 9b3e8b3d-b62b-4434-920f-858f44429596 --similar_to="What is the company's total scope 1 emissions value in 2022?"
```

# Screening

## Questions Types

| Type    | Meaning |
| ------- | ------- |
| metric  | Answer will be structured and provide a numeric value and a unit of measurement |
| boolean | Answer will be a boolean value, either true (Yes) or false (No) | 
| text    | Answer will be simply be a text |

More types will be added later.

## Create Screening

```sh
./scripts/create-screening.sh "$(<< 'EOF'
{
  "questions": [
    {
      "type": "METRIC", 
      "content": "What is the company's total scope 1 emissions value in 2022?"
    },
    {
      "type": "BOOLEAN", 
      "content": "Does the company have a net zero target year?"
    },
    {
      "type": "METRIC", 
      "content": "What is the company's specified net zero target year?"
    }
  ],
  "file_ids": [
    "db0f8303-5bef-415b-b392-f03131853500",
    "bc78aeb5-b8d5-46d1-9e9b-cec8c191bb40"
  ]
}
EOF
)"
```

Screenings are processed asynchronously. Depending on number of files and questions, it can take some time to generate answers. You can use the GET endpoint to poll the API until screening status becomes either `COMPLETED` or `FAILED`. The `./scripts/create-screening.sh` script does this for you.

Example response:

```json
{
  "answers": [
    {
      "evidence": [
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 43,
          "text": "Total Scope 1 for year 2022 is 77476 MTCO2e"
        }
      ],
      "metric": {
        "unit": "MTCO2e",
        "value": 77476
      },
      "question_id": "f3082fbc-be78-4aa9-ac4f-deb6c15dcbb1",
      "text": "The company's total Scope 1 emissions for the year 2022 is 77476 MTCO2e."
    },
    {
      "boolean": true,
      "evidence": [
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 3,
          "text": "By 2050, our goal is to achieve net-zero greenhouse gas emissions, including our financed emissions."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 20,
          "text": "• Achieve net-zero GHG emissions by 2050, including operational emissions (Scope 1 and 2) and emissions attributable to our financing (Scope 3, Category 15)."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 58,
          "text": "We continue to work toward net-zero financed emissions by 2050, and have implemented carbon reduction strategies and purchased renewable energy certificates and carbon offsets sufficient to cover our own Scope 1 and 2 (market-based) emissions."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 13,
          "text": "To facilitate its oversight of climate-related matters, the Corporate Responsibility Committee receives regular updates from our Chief Sustainability Officer and other leaders on matters such as climate-related finance and our goal of achieving net-zero GHG emissions, including financed emissions, by 2050."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 46,
          "text": "As detailed in the Strategy section, in May 2022, we published CO2eMission , our methodology for aligning our financial portfolios with pathways to net-zero by 2050 and for setting interim emissions-based targets to track that alignment."
        }
      ],
      "question_id": "0e1acb69-0091-41b9-a82b-ce7ce95d94bd",
      "text": "Yes, the company has a net-zero target year, which is 2050. The goal is to achieve net-zero greenhouse gas emissions, including financed emissions, by 2050, covering operational emissions (Scope 1 and 2) and emissions attributable to financing (Scope 3, Category 15)."
    },
    {
      "evidence": [
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 3,
          "text": "By 2050, our goal is to achieve net-zero greenhouse gas emissions, including our financed emissions."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 20,
          "text": "• Achieve net-zero GHG emissions by 2050, including operational emissions (Scope 1 and 2) and emissions attributable to our financing (Scope 3, Category 15)."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 58,
          "text": "We continue to work toward net-zero financed emissions by 2050, and have implemented carbon reduction strategies and purchased renewable energy certificates and carbon offsets sufficient to cover our own Scope 1 and 2 (market-based) emissions."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 47,
          "text": "(6) Metric tons of CO2 per metric ton of steel (7) We set our target using the International Energy Agency Net-Zero Emissions by 2050 scenario."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 13,
          "text": "To facilitate its oversight of climate-related matters, the Corporate Responsibility Committee receives regular updates from our Chief Sustainability Officer and other leaders on matters such as climate-related finance and our goal of achieving net-zero GHG emissions, including financed emissions, by 2050."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 47,
          "text": "(9) Our Aviation target – to reduce by 20% the emissions intensity of our Aviation portfolio – is not based on a climate scenario aligned to net zero by 2050."
        },
        {
          "file_id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
          "page": 46,
          "text": "As detailed in the Strategy section, in May 2022, we published CO2eMission , our methodology for aligning our financial portfolios with pathways to net-zero by 2050 and for setting interim emissions-based targets to track that alignment."
        }
      ],
      "metric": {
        "unit": "year",
        "value": 2050
      },
      "question_id": "286c87a2-33c9-4364-ab69-89902c82fa5c",
      "text": "The company's specified net-zero target year is 2050."
    }
  ],
  "created_at": "2025-09-20T22:49:22.631974Z",
  "files": [
    {
      "content_type": "application/pdf",
      "created_at": "2025-09-20T22:48:17.154497Z",
      "extension": "pdf",
      "file_name": "Statement on Emissions.pdf",
      "hash": "65fa6d0a26e38f8a6edee1d6455d90ef6bc4ad11fd80ad69eab30f648fc0e0e4",
      "id": "56a27945-4c74-4bf4-8319-3a094c84e7f0",
      "size": 191945,
      "status": "PROCESSED_SUCCESSFULLY",
      "status_message": "",
      "updated_at": "2025-09-20T22:48:18.48834Z"
    },
    {
      "content_type": "application/pdf",
      "created_at": "2025-09-20T22:48:57.623084Z",
      "extension": "pdf",
      "file_name": "TCFD Report.pdf",
      "hash": "5498dc445bd261b635d47316828c3a0cea48e3af641b0279a667fa1e2a0c0e47",
      "id": "3438f1e8-d97d-4cff-8f6a-4b46b7464d3d",
      "size": 1500685,
      "status": "PROCESSED_SUCCESSFULLY",
      "status_message": "",
      "updated_at": "2025-09-20T22:49:00.033408Z"
    }
  ],
  "id": "3972895d-1656-4451-9f70-38619c90cf6d",
  "questions": [
    {
      "content": "What is the company's total scope 1 emissions value in 2022?",
      "id": "f3082fbc-be78-4aa9-ac4f-deb6c15dcbb1",
      "type": "METRIC"
    },
    {
      "content": "Does the company have a net zero target year?",
      "id": "0e1acb69-0091-41b9-a82b-ce7ce95d94bd",
      "type": "BOOLEAN"
    },
    {
      "content": "What is the company's specified net zero target year?",
      "id": "286c87a2-33c9-4364-ab69-89902c82fa5c",
      "type": "METRIC"
    }
  ],
  "status": "COMPLETED",
  "status_message": "",
  "updated_at": "2025-09-20T22:49:32.365879Z"
}
```

# Testing

In order to run unit and integration tests, just do:

```sh
go test ./... -count=1
```

You need to have docker running as some integration tests use [dockertest](https://github.com/ory/dockertest) to start containers (such as Redis and Postgres).
