#!/bin/bash

set -eu

SCREENING_ID=$1

curl \
    -H 'Content-Type: application/json' \
    http://localhost:8080/screenings/${SCREENING_ID} | jq .
