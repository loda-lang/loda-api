#!/bin/bash

set -euo pipefail

# ensure the container is already running
CONT=$(docker ps -q -f name=loda-api)
if [ -z "$CONT" ]; then
  echo "No running loda-api container found. Aborting."
  exit 1
fi

echo
echo "### COPY SOURCES ###"
GOROOT=/root/go/src/github.com/loda-lang/loda-api/
for f in cmd shared util go.mod go.sum; do
  docker exec loda-api rm -rf $GOROOT/$f
  docker cp $f loda-api:$GOROOT/$f
done
docker cp image/go-build.sh loda-api:/root/
docker cp openapi.v2.yaml loda-api:/data/
docker exec loda-api chmod u+x /root/go-build.sh

echo
echo "### GO BUILD ###"
docker exec loda-api /root/go-build.sh

echo
echo "### NPM BUILD ###"
docker exec -w /root/git/loda-mcp loda-api git pull
docker exec -w /root/git/loda-mcp loda-api npm run build

echo
echo "### CREATE CHECKPOINT ###"
docker exec loda-api curl -sX POST localhost/miner/v1/checkpoint

echo
echo "### RESTART LODA SERVICES ###"
docker exec loda-api /usr/bin/supervisorctl restart programs sequences stats mcp
sleep 1
docker exec loda-api /usr/bin/supervisorctl status

echo
echo "### FINISHED ###"
