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
for f in cmd util go.mod go.sum; do
  docker cp $f loda-api:/root/go/src/github.com/loda-lang/loda-api/
done
docker cp image/go-build.sh loda-api:/root/
docker exec loda-api chmod u+x /root/go-build.sh

echo
echo "### GO BUILD ###"
docker exec loda-api /root/go-build.sh

echo
echo "### FINISHED ###"
