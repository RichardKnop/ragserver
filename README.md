# RAG Server

This project is a generic [RAG](https://cloud.google.com/use-cases/retrieval-augmented-generation?hl=en) server that can be used to answer questions using a konwledge base refined from uploaded PDF documents. In the examples, I use ESG data about scope 1 and scope 2 emissions because that is what I have been testing the server with but it is build to be completely generic and flexible.

When querying the server, you can specify a query type and provide files that will be used to build the context. The server will answer with a structured response depending on a query type as well as list of evidences (files) and specific pages that contain relevant content that was used to generate the answer.

## Table of Contents

- [RAG Server](#rag-server)
  - [Table of Contents](#table-of-contents)
  - [Setup](#setup)
    - [Prerequisites](#prerequisites)
    - [Weaviate](#weaviate)
    - [Database](#database)
    - [Configuration](#configuration)
  - [API](#api)
  - [Adding Documents To Knowledge Base](#adding-documents-to-knowledge-base)
  - [Querying the LLM](#querying-the-llm)
    - [Query Types](#query-types)
    - [Query Request](#query-request)
    - [Metric Query Example](#metric-query-example)

## Setup

### Prerequisites 

Generate HTTP server from OpenAPI spec:

```sh
go generate ./...
```

### Weaviate

To start the Weaviate server:

```sh
docker compose up -d
```

To stop the server:

```sh
docker compose down
```

Delete all objects from the vector database:

```sh
./scripts/weaviate-delete-objects.sh
```

Show all objects in the database:

```sh
./scripts/weaviate-show-all.sh
```

### Database

When you ran the application, it will create a new `db.sqlite` database. You can change the database location by setting `DB_PATH` environment variable.

### Configuration

You can either modify the `config.yaml` file or use environment variables.

| Config               | Meaning |
| -------------------- | --------|
| ai.models.embeddings | Model to use for text embeddings. |
| ai.models.generative | LLM model to use for generating answers. |
| ai.relevant_topics   | Limit scope only to relevant topics when extracting context from PDF files |
| adapter.extract      | Either try to extract context from PDF files locally in the code by using the `pdf` adapter or use Gemini's document vision capability by using the `document` adapter |

## API

See the [OpenAPI spec](/api/api.yaml) for API reference.

Start the server (specify your `GEMINI_API_KEY` env var in .env file):

```sh
source .env
make run
```

## Adding Documents To Knowledge Base

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

## Querying the LLM

### Query Types

| Type   | Meaning |
| ------ | ------- |
| text   | Answer will be simply be a text |
| metric | Answer will be structured and provide a numeric value and a unit of measurement |

More types (such as Yes/No or enum) will be added later.

### Query Request

A query request looks like this:

```json
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e)?", 
    "file_ids": [
      "bd461cce-3c23-4f6a-acb9-e125ebd5ac61",
      "55b42c66-a33e-4811-881f-e35ce2bfd2ac"
    ]
}
```

| Field    | Meaning |
| -------- | ------- |
| type     | Query type. |
| content  | The question you want to ask the LLM. |
| file_ids | Array of file IDs that you want to use as RAG context. |

For content, you could choose some of these example ESG related questions:

1. *What was the company's location-based Scope 2 emissions value (in tCO2e) in 2022?*
2. *What was the company's location-based Scope 2 emissions value (in tCO2e) in 2022?*
3. *What was the company's market-based Scope 2 emissions value (in tCO2e) in 2022?*
4. *What is the company's specified net zero target year in 2022?*

### Metric Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?", 
    "file_ids": [
      "bd461cce-3c23-4f6a-acb9-e125ebd5ac61",
      "55b42c66-a33e-4811-881f-e35ce2bfd2ac"
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
          "file_id": "55b42c66-a33e-4811-881f-e35ce2bfd2ac",
          "page": 3,
          "text": "Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e): Total Scope 1 for year 2022 is 77476"
        },
        {
          "file_id": "bd461cce-3c23-4f6a-acb9-e125ebd5ac61",
          "page": 43,
          "text": "Scope 1 and Scope 2 emissions (location and market based): Total Scope 1 for year 2022 is 77476 MTCO2e"
        }
      ],
      "metric": {
        "unit": "tCO2e",
        "value": 77476
      },
      "text": "The company's Scope 1 emissions in 2022 were 77476 tCO2e."
    }
  ],
  "question": {
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?",
    "file_ids": [
      "bd461cce-3c23-4f6a-acb9-e125ebd5ac61",
      "55b42c66-a33e-4811-881f-e35ce2bfd2ac"
    ],
    "type": "metric"
  }
}
```

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
    "type": "metric"
  }
}
```
