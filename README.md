# RAG Server

What is a RAG server?

> RAG (Retrieval-Augmented Generation) is an AI framework that combines the strengths of traditional information retrieval systems (such as search and databases) with the capabilities of generative large language models (LLMs).

This project is a generic RAG server that can be used to answer questions about any document. However I am mainly using ESG documents to test which is why examples are about scope 1 and 2 emissions, net zero targets etc. 

I have implemented a bit of ESG specific code in the PDF adapter to try to extract yearly tables from the documents (this type of tables often appears in ESG reports). I might eventually remove that part as I am planning to switch to Google's document vision for PDF extraction instead of doing it in code. Or  I might use something like a simple factory pattern to support multiple ways of extracting data from PDFs.

## Table of Contents

- [RAG Server](#rag-server)
  - [Table of Contents](#table-of-contents)
  - [Setup](#setup)
    - [Prerequisites](#prerequisites)
    - [Weaviate](#weaviate)
    - [Database](#database)
  - [API](#api)
  - [Adding Documents To Knowledge Base](#adding-documents-to-knowledge-base)
  - [Querying LLM For Answers](#querying-llm-for-answers)
    - [Metric Query](#metric-query)

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

## Querying LLM For Answers

A query request looks like this:

```json
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e)?", 
    "file_ids": [
        "0fc23ec0-0398-4be2-a266-8eb14a56323f"
    ]
}
```

| Field    | Meaning |
| -------- | ------- |
| type     | Can be either `metric` or `text`. More types will be added later. |
| content  | The question you want to ask the LLM. |
| file_ids | Array of file IDs that you want to use as additional context. |

For content, you could choose some of these example ESG related questions:

1. *What was the company's location-based Scope 2 emissions value (in tCO2e) in 2022?*
2. *What was the company's location-based Scope 2 emissions value (in tCO2e) in 2022?*
3. *What was the company's market-based Scope 2 emissions value (in tCO2e) in 2022?*
4. *What is the company's specified net zero target year in 2022?*

### Metric Query

```sh
./scripts/query.sh "$(<< 'EOF'
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?", 
    "file_ids": [
        "5f354dd1-447b-4cb4-a07d-aebf4ee0a058"
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
      "metric": {
        "unit": "tCO2e",
        "value": 77476
      },
      "text": "The company's Scope 1 emissions value in 2022 was 77476 tCO2e."
    }
  ],
  "question": {
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?",
    "file_ids": [
      "5f354dd1-447b-4cb4-a07d-aebf4ee0a058"
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
