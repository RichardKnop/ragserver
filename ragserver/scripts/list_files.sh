#!/bin/bash

set -eu

# Upload a file to the ragserver
curl \
    -H 'Content-Type: application/json' \
    http://localhost:9020/files/ | jq .
