#!/bin/bash

set -eu

# Upload a file to the ragserver
curl -i \
    -H 'Content-Type: application/json' \
    http://localhost:9020/files/
