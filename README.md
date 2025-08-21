# RAG Server

This project is a generic [RAG](https://cloud.google.com/use-cases/retrieval-augmented-generation?hl=en) server that can be used to answer questions using a konwledge base refined from uploaded PDF documents. In the examples, I use ESG data about scope 1 and scope 2 emissions because that is what I have been testing the server with but it is built to be completely generic and flexible.

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
    - [Boolean Query Example](#boolean-query-example)
    - [Text Query Example](#text-query-example)
    - [No Answer Example](#no-answer-example)

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
