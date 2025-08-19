#!/bin/bash

set -eux

echo '{
  "query": "{
    Get {
      Document { 
        text
        page
        file_id
      }
    }
  }"
}' | tr -d "\n" | curl \
    -X POST \
    -H 'Content-Type: application/json' \
    -d @- \
    http://localhost:9035/v1/graphql | jq .