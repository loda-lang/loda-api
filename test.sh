#!/bin/bash

set -e

echo
echo "### INFLUXDB ###"
curl -fsS http://localhost/influxdb/health
echo
echo

echo "### GRAFANA ###"
curl -fsS http://localhost/grafana/api/health
echo
echo

echo "### OEIS ###"
curl -fsS -O http://localhost/miner/v1/oeis/names.gz && rm names.gz
curl -fsS -O http://localhost/miner/v1/oeis/stripped.gz && rm stripped.gz
curl -fsS -O http://localhost/miner/v1/oeis/b000045.txt.gz && rm b000045.txt.gz
echo

echo "### PROGRAMS ###"
curl -fsS http://localhost/miner/v1/count
curl -fsS http://localhost/miner/v1/session
curl -fsS http://localhost/miner/v1/programs --data-binary $'; Test program 1\nmov $0,26\nadd $0,2\n'
curl -fsS http://localhost/miner/v1/count
curl -fsS http://localhost/miner/v1/programs/0
curl -fsS http://localhost/miner/v1/programs --data-binary $'; Test program 2\nadd $0,17\nsub $0,3\nmul $1,7\n'
curl -fsS http://localhost/miner/v1/count
curl -fsS http://localhost/miner/v1/programs/1
curl -fsS http://localhost/miner/v1/programs --data-binary $'; Test program 3\npow $0,3\n'
curl -fsS http://localhost/miner/v1/count
curl -fsS http://localhost/miner/v1/programs/2
curl -fsS -X POST http://localhost/miner/v1/checkpoint
curl -fsS http://localhost/miner/v1/count
curl -fsS http://localhost/miner/v1/programs/0
curl -fsS http://localhost/miner/v1/programs/1
curl -fsS http://localhost/miner/v1/programs/2

echo "### STATS ###"
curl -fsS -X POST http://localhost/miner/v1/cpuhours
