# RAG (Retrieval-Augmented Generation) Server

> RAG (Retrieval-Augmented Generation) is an AI framework that combines the strengths of traditional information retrieval systems (such as search and databases) with the capabilities of generative large language models (LLMs).

## Setup

You can skip this step as images are not yet supported so OCR is not needed.

~~You will need to install [Tesseract](https://github.com/tesseract-ocr/tessdoc) C++ OCR library.~~

```sh
brew install tesseract
```

### Weaviate

To start the Weaviate server:

```sh
docker compose up -d
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

## RAG Server

Start the server (specify your `GEMINI_API_KEY` env var in .env file):

```sh
cd ragserver
source .env
make run
```

### Adding Data To Knowledge Base

Upload PDF files which will be used to extract documents:

```sh
./scripts/upload-file.sh '/Users/richardknop/Desktop/Statement on Emissions.pdf'
./scripts/upload-file.sh '/Users/richardknop/Desktop/TCFD Report.pdf'
```

Keep track of file IDs because those are required to query the LLM for an answer.

You can list all current files:

```sh
./scripts/list_files.sh
```

### Querying LLM For Answers

#### Request

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

#### Example

For content, you could choose some of these example ESG related questions:

1. *What was the company's location-based Scope 2 emissions value (in tCO2e)?*
2. *What was the company's location-based Scope 2 emissions value (in tCO2e)?*
3. *What was the company's market-based Scope 2 emissions value (in tCO2e)?*
4. *What is the company's specified net zero target year?*

You also probably want to specify year. For example:

```sh
./scripts/query.sh "$(<< 'EOF'
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e) in 2022?", 
    "file_ids": [
        "0fc23ec0-0398-4be2-a266-8eb14a56323f"
    ]
}
EOF
)"
```
