#!/bin/bash

set -eu

# Check if an argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 '<your file>'"
    exit 1
fi

# Upload a file to the ragserver and capture the uploaded file ID
FILE=$1

file_id=$(curl -X POST \
    -H 'Content-Type: multipart/form-data' \
    -F file=@"$FILE" \
    http://localhost:8080/files -s | jq -r ".id");

printf "\nUploading a file with ID $file_id\n"

file_url="http://localhost:8080/files/$file_id"
interval_in_seconds=1
status_path=".status"

printf "\nPolling '${file_url%\?*}' every $interval_in_seconds seconds, until processing successful or failed\n"

while true;
do
    status=$(curl -H 'Content-Type: application/json' $file_url | jq -r $status_path);
    printf "\r$(date +%H:%M:%S): $status";
    if [[ "$status" == "PROCESSED_SUCCESSFULLY" || "$status" == "PROCESSING_FAILED" ]]; then
        curl \
            -H 'Content-Type: application/json' \
            ${file_url} | jq .
        break;
    fi;
    sleep $interval_in_seconds;
done
