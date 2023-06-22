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

function ensure_dir {
  echo "Ensuring directory $1:$2"
  if ! gssh "$1" "test -d \$HOME/$2"; then
    echo "Error: command failed or directory not found: $1:$2"
    exit 1
  fi
}

function forbid_dir {
  echo "Forbidding directory $1:$2"
  if ! gssh "$1" "! test -d \$HOME/$2"; then
    echo "Error: command failed or directory exists already: $1:$2"
    exit 1
  fi
}

ensure_dir $source_host data
ensure_dir $source_host grafana
ensure_dir $source_host influxdb
forbid_dir $source_host influxdb-backup

forbid_dir $target_host data
forbid_dir $target_host grafana
forbid_dir $target_host influxdb

echo "Creating checkpoint"
gssh $source_host "curl -X POST localhost/miner/v1/checkpoint"

echo "Creating backup"
gssh $source_host "docker exec loda-api /usr/bin/influxd backup -portable /influxdb-backup"
gssh $source_host "docker cp loda-api:/influxdb-backup ."
gssh $source_host "docker exec loda-api rm -R /influxdb-backup"
ensure_dir $source_host influxdb-backup

echo "Fetching data"
gscp "$source_host:data/checkpoint.txt" .
gscp "$source_host:data/setup.txt" .
gscp "$source_host:grafana" .
# gscp "$source_host:influxdb-backup" .

echo "Pushing data"
gssh $target_host "mkdir -p \$HOME/data"
gscp checkpoint.txt $target_host:data/
gscp setup.txt $target_host:data/
gscp grafana $target_host

# [ -n "$(docker ps -q -f name=loda-api)" ]
