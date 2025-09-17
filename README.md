# RAG Server

- [RAG Server](#rag-server)
  - [Components](#components)
    - [Extractor](#extractor)
    - [Embedder](#embedder)
    - [Retriever](#retriever)
    - [GenerativeModel](#generativemodel)
- [Examples](#examples)
- [SQLite Database](#sqlite-database)
- [Configuration](#configuration)
- [API](#api)
- [Adding Documents To Knowledge Base](#adding-documents-to-knowledge-base)
- [Screening](#screening)
  - [Questions Types](#questions-types)
  - [Create Screening](#create-screening)
  - [Metric Query Example](#metric-query-example)
  - [Boolean Query Example](#boolean-query-example)
  - [Text Query Example](#text-query-example)
  - [No Answer Example](#no-answer-example)
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
	Extract(ctx context.Context, contents io.ReadSeeker, topics RelevantTopics) ([]Document, error)
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
  storeAdapter
)
restAdapter := rest.New(rs)
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

Redis retriever backend, local PDF extractor, Gemini text embedding and generative model:

```sh
docker compose -f examples/redis/docker-compose.yml up -d
```

Weaviate retriever backend, local PDF extractor, Gemini text embedding and generative model:

```sh
docker compose -f examples/weaviate/docker-compose.yml up -d
```

Redis retriever backend, local PDF extractor, hugot open source text embedding and Gemini generative model:

```sh
docker compose -f examples/redis-hugot/docker-compose.yml up -d
```

You need to have `GEMINI_API_KEY` environment variable set for the examples to work.

# SQLite Database

This project uses sqlite as simple embedded SQL database. It is used to store information about uploaded files including file size, hash, content type etc. UUIDs from the SQL database should be referenced in the weaviate database as a `file_id` property.

When you ran the application, it will create a new `db.sqlite` database. You can change the database file by setting `DB_NAME` environment variable.

# Configuration

This project requires a Gemini API key. Use `GEMINI_API_KEY` environment variable to set your API key.

For everything else, you can use whatever configuration method you prefer. If you use one of the examples, they rely on [viper](https://github.com/spf13/viper) to read configuration from a YAML config file.

The example `config.example.yaml` defines many different options:

| Config                    | Meaning |
| ------------------------- | --------|
| adapter.extract.name      | Either try to extract context from PDF files locally in the code by using the `pdf` adapter or use Gemini's document vision capability by using the `document` adapter |
| adapter.extract.model     | Only used if `adapter.extract.name` is set to `document`. Currently only supported model is `gemini-2.5-flash` |
| adapter.embed.name        | Currently supported are `google-genai` and `hugot` |
| adapter.embed.model       | Set to `text-embedding-004` for `google-genai` adapter and `sentence-transformers/all-MiniLM-L6-v2` for `hugot` adapter |
| adapter.retrieve.name     | Supported adapters are `weaviate` and `redis` |
| redis.vector_dim          | If you are using Redis, set to 768 for `text-embedding-004` or 384 for `sentence-transformers/all-MiniLM-L6-v2` |
| adapter.generative.name   | Currently supported are `google-genai` and `hugot` |
| adapters.generative.model | Set to `gemini-2.5-flash` for `google-genai` adapter and `onnx-community/gemma-3-270m-it-ONNX` for `hugot` adapter |
| relevant_topics           | Limit scope only to relevant topics when extracting context from PDF files |

You can set any configuration value by using `_` as env key replacer. For example, a `http.host` can be set as environment variable `HTTP_HOST` and so on.

For local testing, I suggest switching `adapter.extract` from `document` to `pdf`. Document processing by Gemini model is a bit expensive so if you are uploading lots of files during development, using the `pdf` adapter and only doing final end to end checks with `document` adapter will be more cost efficient.

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

You can also list documents extracted from a specific file (currently limited to 100 documents, no pagination support):

```sh
./scripts/list-file-documents.sh bc4509b4-c156-4478-890d-8d98a44abf03
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

NOTE: Screenings are still work in progress - skip to next example for now.

```sh
./scripts/create-screening.sh "$(<< 'EOF'
{
  "questions": [
    {
      "type": "METRIC", 
      "content": "What is the company's total scope 1 emissions value in 2022?"
    },
    {
      "type": "METRIC", 
      "content": "What is the company's specified net zero target year in 2022?"
    }
  ],
  "file_ids": [
    "d89263ab-3954-4acb-952b-7303d0151f7b",
    "2d3e475a-493e-4b88-ab11-3de8ab74e59a"
  ]
}
EOF
)"
```

Response:

```json
{
  "answers": [],
  "created_at": "2025-09-17T16:25:28.922341+01:00",
  "files": [
    {
      "content_type": "application/pdf",
      "created_at": "2025-09-17T15:16:49.719Z",
      "extension": "pdf",
      "file_name": "Statement on Emissions.pdf",
      "hash": "65fa6d0a26e38f8a6edee1d6455d90ef6bc4ad11fd80ad69eab30f648fc0e0e4",
      "id": "d89263ab-3954-4acb-952b-7303d0151f7b",
      "size": 191945,
      "status": "PROCESSED_SUCCESSFULLY",
      "status_message": "",
      "updated_at": "2025-09-17T15:16:50.79Z"
    },
    {
      "content_type": "application/pdf",
      "created_at": "2025-09-17T15:17:03.903Z",
      "extension": "pdf",
      "file_name": "TCFD Report.pdf",
      "hash": "5498dc445bd261b635d47316828c3a0cea48e3af641b0279a667fa1e2a0c0e47",
      "id": "2d3e475a-493e-4b88-ab11-3de8ab74e59a",
      "size": 1500685,
      "status": "PROCESSED_SUCCESSFULLY",
      "status_message": "",
      "updated_at": "2025-09-17T15:17:05.698Z"
    }
  ],
  "id": "d5f54b05-a75c-4357-bd0b-137a5623c761",
  "questions": [
    {
      "content": "What is the company's total scope 1 emissions value in 2022?",
      "id": "373a4e5b-66b8-4964-8c70-305e6f0818ec",
      "type": "METRIC"
    },
    {
      "content": "What is the company's specified net zero target year in 2022?",
      "id": "2d0a4f73-04bb-4497-b771-b45b52710c21",
      "type": "METRIC"
    }
  ],
  "status": "REQUESTED",
  "status_message": "",
  "updated_at": "2025-09-17T16:25:28.922341+01:00"
}
```

## Metric Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
  "question": {
    "type": "METRIC", 
    "content": "What is the company's total scope 1 emissions value in 2022?"
  },
  "file_ids": [
    "67817687-8652-4734-af0a-7962955bce4a",
    "e6cdec78-1e35-46fc-a175-7764e71903ba"
  ]
}
EOF
)"
```

Example response:

```json
{
  "answers": [
    {
      "evidence": [
        {
          "file_id": "73ad3166-1627-4b7e-82a3-31427ad5444e",
          "page": 43,
          "text": "Total Scope 1 emissions: 86,602 (2019 baseline), 78,087 (2020), 73,319* (2021), 77,476* (2022)."
        },
        {
          "file_id": "67224b92-bb64-457d-8cfc-584539292c5c",
          "page": 3,
          "text": "For Scope 1 and Scope 2 (location & market based) Emissions in 2022: Total Scope 1 was 77,476; Total Scope 2 (location) was 593,495; Total Scope 2 (market) was 4,424; Total Scope 1 and 2 (location) was 670,972; Total Scope 1 and 2 (market) was 81,901."
        }
      ],
      "metric": {
        "unit": "tCO2e",
        "value": 77476
      },
      "text": "The company's Scope 1 emissions value in 2022 was 77,476 tCO2e."
    }
  ],
  "question": {
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?",
    "file_ids": [
      "67224b92-bb64-457d-8cfc-584539292c5c",
      "73ad3166-1627-4b7e-82a3-31427ad5444e"
    ],
    "type": "METRIC"
  }
}
```

## Boolean Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
  "question": {
    "type": "BOOLEAN", 
    "content": "Does the company have a net zero target year?"
  },
  "file_ids": [
    "67224b92-bb64-457d-8cfc-584539292c5c",
    "73ad3166-1627-4b7e-82a3-31427ad5444e"
  ]
}
EOF
)"
```

Example response:

```json
{
  "answers": [
    {
      "boolean": true,
      "evidence": [
        {
          "file_id": "73ad3166-1627-4b7e-82a3-31427ad5444e",
          "page": 28,
          "text": "Since announcing its net-zero GHG emissions goal by 2050 (including financed emissions) in March 2021, the bank disclosed first interim targets in May 2022 for Oil & Gas and Power sectors, based on a 2019 baseline."
        },
        {
          "file_id": "73ad3166-1627-4b7e-82a3-31427ad5444e",
          "page": 18,
          "text": "These include: The Climate Steering Committee, chaired by the Chief Sustainability Officer, which provides strategic direction and monitors progress on climate-related goals (net-zero GHG emissions by 2050, $500 billion sustainable finance by 2030, Institute for Sustainable Finance establishment) and includes members from all principal lines of business and impacted functions."
        },
        {
          "file_id": "73ad3166-1627-4b7e-82a3-31427ad5444e",
          "page": 20,
          "text": "Core climate-related goals are: deploying $500 billion in sustainable finance by 2030 (environmental and social finance), and achieving net-zero GHG emissions by 2050 (including operational Scope 1 and 2, and financed Scope 3, Category 15 emissions)."
        },
        {
          "file_id": "73ad3166-1627-4b7e-82a3-31427ad5444e",
          "page": 46,
          "text": "In May 2022, CO2eMission methodology was published, aligning financial portfolios with net-zero pathways by 2050 and setting interim targets for Oil & Gas and Power sectors."
        }
      ],
      "text": "Yes, the company has announced a net-zero GHG emissions goal by 2050."
    }
  ],
  "question": {
    "content": "Does the company have a net zero target year?",
    "file_ids": [
      "67224b92-bb64-457d-8cfc-584539292c5c",
      "73ad3166-1627-4b7e-82a3-31427ad5444e"
    ],
    "type": "BOOLEAN"
  }
}
```

## Text Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
  "question": {
    "type": "TEXT", 
    "content": "What is the company's specified net zero target year?"
  },
  "file_ids": [
    "67224b92-bb64-457d-8cfc-584539292c5c",
    "73ad3166-1627-4b7e-82a3-31427ad5444e"
  ]
}
EOF
)"
```

Example response:

```json
{
  "answers": [
    {
      "evidence": [
        {
          "file_id": "dde50d1a-e474-43f3-843d-3f37dc81cc97",
          "page": 58,
          "text": "The company aims to continue working toward net-zero financed emissions by 2050, having already implemented carbon reduction strategies and offset its own Scope 1 and 2 (market-based) emissions."
        },
        {
          "file_id": "dde50d1a-e474-43f3-843d-3f37dc81cc97",
          "page": 31,
          "text": "They joined the \"Net-Zero Banking Alliance (NZBA)\" in October 2021, an industry-led group aiming to align bank financing with net-zero GHG emissions by mid-century."
        },
        {
          "file_id": "dde50d1a-e474-43f3-843d-3f37dc81cc97",
          "page": 46,
          "text": "In May 2022, they published CO2eMission, their methodology for aligning financial portfolios with net-zero pathways by 2050 and setting interim targets."
        }
      ],
      "text": "The company's specified net zero target year is 2050. This goal is for net-zero financed emissions and aligning financial portfolios with net-zero pathways, consistent with their membership in the Net-Zero Banking Alliance which aims for net-zero GHG emissions by mid-century."
    }
  ],
  "question": {
    "content": "What is the company's specified net zero target year?",
    "file_ids": [
      "67224b92-bb64-457d-8cfc-584539292c5c",
      "73ad3166-1627-4b7e-82a3-31427ad5444e"
    ],
    "type": "TEXT"
  }
}
```

## No Answer Example 

If you ask a question model cannot answer from the provided context, it will simply return an empty answer

```json
{
  "answers": [
    {
      "evidence": [],
      "metric": {
        "unit": "",
        "value": 0
      },
      "text": ""
    }
  ],
  "question": {
    "content": "What color is my hair?",
    "file_ids": [
      "5f354dd1-447b-4cb4-a07d-aebf4ee0a058"
    ],
    "type": "METRIC"
  }
}
```

# Testing

In order to run unit and integration tests, just do:

```sh
go test ./... -count=1
```

You need to have docker running as some integration tests use [dockertest](https://github.com/ory/dockertest) to start containers (such as Redis).
