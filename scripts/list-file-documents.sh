#!/bin/bash

set -eu

# Capture the query from the command-line argument
FILE_ID=$1

# Upload a file to the ragserver
curl \
    -H 'Content-Type: application/json' \
    http://localhost:9020/files/${FILE_ID}/documents | jq .
