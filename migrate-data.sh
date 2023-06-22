#!/bin/bash

set -euo pipefail

for cmd in gcloud; do
  if ! [ -x "$(command -v $cmd)" ]; then
    echo "Error: $cmd is not installed" >&2
    exit 1
  fi
done

if [ "$#" -ne 4 ]; then
  echo "Usage: $0 <zone> <project> <source-host> <target-host>"
  exit 1
fi

zone=$1
project=$2
source_host=$3
target_host=$4

if [ "$source_host" == "$target_host" ]; then
  echo "Error: source host equals target host"
  exit 1
fi

function gssh {
  gcloud beta compute ssh --zone "$zone" --project "$project" "$1" --command "$2"
}

function gscp {
  gcloud beta compute scp --zone "$zone" --project "$project" --recurse "$1" "$2"
}

data_dirs=data grafana influxdb

for dir in $data_dirs; do
  echo "Checking directory $source_host:$dir"
  if ! gssh "$source_host" "test -d \$HOME/$dir"; then
    echo "Error: command failed or directory not found: $source_host:$dir"
    exit 1
  fi
done

for dir in $data_dirs; do
  echo "Checking directory $target_host:$dir"
  if ! gssh "$target_host" "! test -d \$HOME/$dir"; then
    echo "Error: command failed or directory exists already: $target_host:$dir"
    exit 1
  fi
done

echo "Creating checkpoint"
gssh "$source_host" "curl -X POST localhost/miner/v1/checkpoint" || true

for dir in $data_dirs; do
  gscp $source_host:$dir .
done

for dir in $data_dirs; do
  gscp $dir $target_dir
done

# [ -n "$(docker ps -q -f name=loda-api)" ]
