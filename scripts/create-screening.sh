#!/bin/bash

set -eu

# Check if an argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 '<your file>'"
    exit 1
fi

# Upload a file to the ragserver and capture the uploaded file ID
PAYLOAD=$1

screening_id=$(echo "$PAYLOAD" | curl \
    -X POST \
    -H 'Content-Type: application/json' \
    -d @- \
    http://localhost:8080/screenings -s | jq -r ".id");

printf "\nCreated a screening with ID $screening_id\n"

screening_url="http://localhost:8080/screenings/$screening_id"
interval_in_seconds=1
status_path=".status"

printf "\nPolling '${screening_url%\?*}' every $interval_in_seconds seconds, until successful or failed\n"

while true;
do
    status=$(curl -H 'Content-Type: application/json' $screening_url | jq -r $status_path);
    printf "\r$(date +%H:%M:%S): $status";
    if [[ "$status" == "SUCCESSFUL" || "$status" == "FAILED" ]]; then
        curl \
            -H 'Content-Type: application/json' \
            ${screening_url} | jq .
        break;
    fi;
    sleep $interval_in_seconds;
done
