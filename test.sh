#!/bin/bash

set -eo pipefail

local_mode=false

if [ "$local_mode" = true ]; then
    echo "### LOCAL MODE ###"
    sequences_host_v1=http://localhost:8080/v1
    sequences_host_v2=http://localhost:8080/v2
    programs_host_v1=http://localhost:8081/v1
    programs_host_v2=http://localhost:8081/v2
    stats_host_v1=http://localhost:8082/v1
    stats_host_v2=http://localhost:8082/v2
else
    echo "### SERVER MODE ###"
    sequences_host_v1=http://localhost/v1/miner
    sequences_host_v2=http://localhost/v2
    programs_host_v1=http://localhost/v1/miner
    programs_host_v2=http://localhost/v2
    stats_host_v1=http://localhost/v1/miner
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
echo "### SEQUENCES V1 ###"
curl -fsS -O ${sequences_host_v1}/oeis/names.gz && rm names.gz
curl -fsS -O ${sequences_host_v1}/oeis/stripped.gz && rm stripped.gz
curl -fsS -O ${sequences_host_v1}/oeis/b000045.txt.gz && rm b000045.txt.gz
echo
echo "### SEQUENCES V2 ###"
curl -fsS ${sequences_host_v2}/sequences/A000045
curl -fsS ${sequences_host_v2}/sequences/search?q=Fibonacci
curl -X POST --data-binary @testdata/programs/A000042.asm -H "Content-Type: text/plain" ${programs_host_v2}/programs/eval

echo
echo "### PROGRAMS V1 ###"
curl -fsS ${programs_host_v1}/count
curl -fsS ${programs_host_v1}/session
curl -fsS ${programs_host_v1}/programs --data-binary $'; Test program 1\nmov $0,26\nadd $0,2\n'
curl -fsS ${programs_host_v1}/count
curl -fsS ${programs_host_v1}/programs/0
curl -fsS ${programs_host_v1}/programs --data-binary $'; Test program 2\nadd $0,17\nsub $0,3\nmul $1,7\n'
curl -fsS ${programs_host_v1}/count
curl -fsS ${programs_host_v1}/programs/1
curl -fsS ${programs_host_v1}/programs --data-binary $'; Test program 3\npow $0,3\n'
curl -fsS ${programs_host_v1}/count
curl -fsS ${programs_host_v1}/programs/2
curl -fsS -X POST ${programs_host_v1}/checkpoint
curl -fsS ${programs_host_v1}/count
curl -fsS ${programs_host_v1}/programs/0
curl -fsS ${programs_host_v1}/programs/1
curl -fsS ${programs_host_v1}/programs/2

echo
echo "### PROGRAMS V2 ###"
curl -fsS ${programs_host_v2}/programs/A000045
curl -fsS ${programs_host_v2}/programs/search?q=Fibonacci

echo
echo "### STATS V1 ###"
curl -fsS -X POST ${stats_host_v1}/cpuhours

echo
echo "### STATS V2 ###"
curl -fsS -O ${stats_host_v2}/stats/submitters && rm submitters
