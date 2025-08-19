#!/bin/bash

# This script interacts with a running ragserver to add some example documents.

set -eux

echo '{
	"documents": [
	{"text": "TDXIRV is an environment variable for controlling throttle speed"},
	{"text": "some flags for setting acceleration are --accelxyzp and --acceljjrv"},
	{"text": "acceleration is also affected by the ACCUVI5 env var"},
	{"text": "/usr/local/fuel555 contains information about fuel capacity"},
	{"text": "we can control fuel savings with the --savemyfuelplease flag"},
	{"text": "fuel savings can be observed on local port 48332"},
	{"text": "Total Scope 1 77,476"},
	{"text": "Total Scope 2 (location) 593,495"},
	{"text": "Total Scope 2 (market)2 4,424"},
	{"text": "Total Scope 1 and 2 (location) 670,972"},
	{"text": "Total Scope 1 and 2 (market) 81,901"},
    {"text": "The Board's Corporate Responsibility Committee assists the Board in fulfilling its responsibilities to oversee the Company's significant strategies, policies, and programs on social and public responsibility matters, including environmental sustainability and climate change. To facilitate its oversight of climate-related matters, the Corporate Responsibility Committee receives regular updates from our Chief Sustainability Officer and other leaders on matters such as climate-related finance and our goal of achieving net-zero GHG emissions, including financed emissions, by 2050. Other forms of Board engagement with climate-related risks and opportunities occur as needed and will continue to evolve as needs dictate."}
]}' | tr -d "\n" | curl \
		-X POST \
    -H 'Content-Type: application/json' \
    -d @- \
    http://localhost:9020/documents