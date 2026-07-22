#!/bin/bash

set -eo pipefail

local_mode=false

if [ "$local_mode" = true ]; then
    echo "### LOCAL MODE ###"
    sequences_host_v2=http://localhost:8080/v2
    programs_host_v2=http://localhost:8081/v2
    submissions_host_v2=http://localhost:8084/v2
    stats_host_v2=http://localhost:8082/v2
else
    echo "### SERVER MODE ###"
    sequences_host_v2=http://localhost/v2
    programs_host_v2=http://localhost/v2
    submissions_host_v2=http://localhost/v2
    stats_host_v2=http://localhost/v2
fi

if [ "$local_mode" != true ]; then
    echo
    echo "### INFLUXDB ###"
    curl -fsS http://localhost/influxdb/health
    echo
    echo "### GRAFANA ###"
    curl -fsS http://localhost/grafana/api/health
fi

echo
echo "### SEQUENCES V2 ###"
curl -fsS ${sequences_host_v2}/sequences/A000045
curl -fsS ${sequences_host_v2}/sequences/search?q=Fibonacci
curl -X POST --data-binary @testdata/programs/A000042.asm -H "Content-Type: text/plain" ${programs_host_v2}/programs/eval

echo
echo "### PROGRAMS V2 ###"
curl -fsS ${programs_host_v2}/programs/A000045
curl -fsS ${programs_host_v2}/programs/search?q=Fibonacci

echo
echo "### STATS V2 ###"
curl -fsS -O ${stats_host_v2}/stats/submitters && rm submitters
