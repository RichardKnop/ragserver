#!/bin/bash

set -eu

FILE_ID=$1

curl -X DELETE \
    -H 'Content-Type: application/json' \
    http://localhost:8080/files/${FILE_ID} 
