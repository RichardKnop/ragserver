# RAG Server 

## Setup

You will need to install [Tesseract](https://github.com/tesseract-ocr/tessdoc) C++ OCR library.

```sh
brew install tesseract
```

## Weaviate

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

## RAG Server

Start the server (specify your GEMINI_API_KEY env var in .env file):

```sh
cd ragserver
source .env
go run .
```

### Adding Data To Knowledge Base

You can either add some text documents:

```sh
./scripts/add-documents.sh
```

Or upload a PDF file which will be used to extract documents:

```sh
./scripts/upload-file.sh '/Users/richardknop/Desktop/Statement on Emissions.pdf'
./scripts/upload-file.sh '/Users/richardknop/Desktop/TCFD Report.pdf'
```

### Querying LLM For Answers

Query for scope 1 emissions:

```sh
export QUERY="What was the company's Scope 1 emissions value (in tCO2e)?"
export PAYLOAD=$(echo "{\"type\": \"metric\", \"content\": \"$QUERY\"}")
./scripts/query.sh "$PAYLOAD"
```

Query for scope 2 location based emissions:

```sh
export QUERY="What was the company's location-based Scope 2 emissions value (in tCO2e)?"
export PAYLOAD=$(echo "{\"type\": \"metric\", \"content\": \"$QUERY\"}")
./scripts/query.sh "$PAYLOAD"
```

Query for scope 2 market based emissions:

```sh
export QUERY="What was the company's market-based Scope 2 emissions value (in tCO2e)?"
export PAYLOAD=$(echo "{\"type\": \"metric\", \"content\": \"$QUERY\"}")
./scripts/query.sh "$PAYLOAD"
```

Query for net zero target:

```sh
export QUERY="What is the company's specified net zero target year?"
export PAYLOAD=$(echo "{\"type\": \"metric\", \"content\": \"$QUERY\"}")
./scripts/query.sh "$PAYLOAD"
```
