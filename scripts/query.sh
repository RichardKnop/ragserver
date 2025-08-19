#!/bin/bash

set -eu

# Check if an argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 '<your query>'"
    exit 1
fi

set -x

# Capture the query from the command-line argument
PAYLOAD=$1

# Send the request
echo "$PAYLOAD" | tr -d "\n" | curl \
    -X POST \
    -H 'Content-Type: application/json' \
    -d @- \
    http://localhost:9020/query | jq .