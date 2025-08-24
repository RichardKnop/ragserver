# RAG Server

This project is a generic [RAG](https://cloud.google.com/use-cases/retrieval-augmented-generation?hl=en) server that can be used to answer questions using a knowledge base (corpus) refined from uploaded PDF documents. In the examples, I use ESG data about scope 1 and scope 2 emissions because that is what I have been testing the server with but it is built to be completely generic and flexible.

When querying the server, you can specify a query type and provide files that should contain the answer. The server uses embedding model to get a vector representation of the question and retrieve documents from the knowledge base that are most similar to the question. It will then generate a structured JSON answer depending on a query type as well as list of evidences (files) and specific pages in PDFs referencing where the answer was extracted from.

NOTE: I am planning to refactor this project as library so it can be imported from external repositories. For now everything is in `internal` but once I feel it works well enough I will refactor this into a library.

## Table of Contents

- [RAG Server](#rag-server)
  - [Table of Contents](#table-of-contents)
  - [Setup](#setup)
    - [Prerequisites](#prerequisites)
    - [Vector Database](#vector-database)
    - [Database](#database)
    - [Configuration](#configuration)
  - [API](#api)
  - [Adding Documents To Knowledge Base](#adding-documents-to-knowledge-base)
  - [Querying the LLM](#querying-the-llm)
    - [Query Types](#query-types)
    - [Query Request](#query-request)
    - [Metric Query Example](#metric-query-example)
    - [Boolean Query Example](#boolean-query-example)
    - [Text Query Example](#text-query-example)
    - [No Answer Example](#no-answer-example)
  - [Testing](#testing)

## Setup

### Prerequisites 

Generate HTTP server from OpenAPI spec:

```sh
go generate ./...
```

### Vector Database

You can choose between [weaviate](https://github.com/weaviate/weaviate) and [redis](https://redis.io/) as a vector database.

Currently the only supported text embedding models (`text-embedding-004`) use 768 dimensional vector space, other number of dimensions will not work with redis adapter as its index is hardcoded to 768 dimensions. I am looking into making it more dynamic, perhaps creating different indexes based on embedding model.

Use docker compose to start a weaviate or redis database:

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

This project uses sqlite as simple embedded SQL database. It is used to store information about uploaded files including file size, hash, content type etc. UUIDs from the SQL database should be referenced in the weaviate database as a `file_id` property.

When you ran the application, it will create a new `db.sqlite` database. You can change the database file by setting `DB_NAME` environment variable and migrations folder by setting `DB_MIGRATIONS_PATH` environment variable.

### Configuration

This project requires a Gemini API key. Use `GEMINI_API_KEY` environment variable to set your API key.

You can either modify the `config.yaml` file or use environment variables.

| Config                  | Meaning |
| ----------------------- | --------|
| adapter.extract.name    | Either try to extract context from PDF files locally in the code by using the `pdf` adapter or use Gemini's document vision capability by using the `document` adapter |
| adapter.extract.model   | Only used if `adapter.extract.name` is set to `document`. Currently only supported model is `gemini-2.5-flash` |
| adapter.embed.name      | Currently supported are `google-genai` and `hugot` . Set `models.embeddings` to `text-embedding-004` for `google-genai` and `all-MiniLM-L6-v2` for `hugot` |
| adapter.embed.model     | Model to use for text embeddings. Currently supported are Gemini's `text-embedding-004` and OONX `all-MiniLM-L6-v2`. |
| adapter.retrieve.name   | Supported adapters are `weaviate` and `redis` |
| adapter.generative.name | Currently only supported generative model is `gemini-2.5-flash` |
| redis.vector_dim        | If you are using Redis, set to 768 for `text-embedding-004` or 384 for `all-MiniLM-L6-v2` |
| models.generative.name  | LLM model to use for generating answers. Currently only Gemini models supported such as `gemini-2.5-flash`. |
| relevant_topics         | Limit scope only to relevant topics when extracting context from PDF files |

There is more configuration that can be referenced via `config.yaml` file. You can set any configuration value by using `_` as env key replacer. For example, a `http.host` can be set as environment variable `HTTP_HOST` and so on.

For local testing, I suggest switching `adapter.extract` from `document` to `pdf`. Document processing by Gemini model is a bit expensive so if you are uploading lots of files during development, using the `pdf` adapter and only doing final end to end checks with `document` adapter will be more cost efficient.

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

You can also list documents extracted from a specific file (currently limited to 100 documents, no pagination support):

```sh
./scripts/list-file-documents.sh bc4509b4-c156-4478-890d-8d98a44abf03
```

## Querying the LLM

### Query Types

| Type    | Meaning |
| ------- | ------- |
| metric  | Answer will be structured and provide a numeric value and a unit of measurement |
| boolean | Answer will be a boolean value, either true (Yes) or false (No) | 
| text    | Answer will be simply be a text |

More types will be added later.

### Query Request

An example query request looks like this:

```json
{
    "type": "metric", 
    "content": "What was the company's Scope 1 emissions value (in tCO2e)?", 
    "file_ids": [
      "67224b92-bb64-457d-8cfc-584539292c5c",
      "73ad3166-1627-4b7e-82a3-31427ad5444e"
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
      "9b124afe-8dd8-46a5-820b-d172e4fd90e6",
      "043244c1-af65-4bad-8d90-969b0d8698d2"
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
    "type": "metric"
  }
}
```

### Boolean Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
    "type": "boolean", 
    "content": "Does the company have a net zero target year?", 
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
    "type": "boolean"
  }
}
```

### Text Query Example

```sh
./scripts/query.sh "$(<< 'EOF'
{
    "type": "text", 
    "content": "What is the company's specified net zero target year?", 
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
    "type": "text"
  }
}
```

### No Answer Example 

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

## Testing

In order to run unit and integration tests, just do:

```sh
go test ./... -count=1
```

You need to have docker running as some integration tests use [dockertest](https://github.com/ory/dockertest) to start containers (such as Redis).
