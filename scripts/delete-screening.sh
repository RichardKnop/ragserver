#!/bin/bash

set -eu

SCREENING_ID=$1

curl -X DELETE \
    -H 'Content-Type: application/json' \
    http://localhost:8080/screenings/${SCREENING_ID} 
