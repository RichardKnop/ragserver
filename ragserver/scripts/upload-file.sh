#!/bin/bash

set -eu

# Check if an argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 '<your file>'"
    exit 1
fi

set -x

# Capture the file path from the command-line argument
FILE=$1

# Upload a file to the ragserver
curl -i \
    -X POST \
    -H 'Content-Type: multipart/form-data' \
    -F file=@"$FILE" \
    http://localhost:9020/files/
