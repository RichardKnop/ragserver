#!/bin/bash

set -eu

curl \
    -H 'Content-Type: application/json' \
    http://localhost:8080/screenings | jq .
