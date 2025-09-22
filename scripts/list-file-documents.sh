#!/bin/bash

set -eu

FILE_ID=$1
SIMILAR_TO=""

while [ $# -gt 0 ]; do
  case "$1" in
    --similar_to=*)
      SIMILAR_TO="${1#*=}"
      ;;
  esac
  shift
done

curl -X GET -G \
    -H 'Content-Type: application/json' \
    --data-urlencode "similar_to=${SIMILAR_TO}" \
    --data-urlencode "limit=10" \
    http://localhost:8080/files/${FILE_ID}/documents | jq .
