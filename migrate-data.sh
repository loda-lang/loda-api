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

dryrun=true

if [ "$source_host" = "$target_host" ]; then
  echo "Error: source host equals target host"
  exit 1
fi

function gssh {
  echo ">>> gcloud beta compute ssh --zone $zone --project $project $1 --command \"$2\""
  if [ -z "$dryrun" ]; then
    gcloud beta compute ssh --zone "$zone" --project "$project" "$1" --command "$2"
  fi
}

function gscp {
  echo ">>> gcloud beta compute scp --zone $zone --project $project --recurse $1 $2"
  if [ -z "$dryrun" ]; then
    gcloud beta compute scp --zone "$zone" --project "$project" --recurse "$1" "$2"
  fi
}

function rm_local {
  echo ">>> rm -R $@"
  if [ -z "$dryrun" ]; then
    rm -R $@
  fi
}

function ensure_dir {
  if ! gssh "$1" "test -d \$HOME/$2"; then
    echo "Error: command failed or directory not found: $1:$2"
    exit 1
  fi
}

function ensure_file {
  if ! gssh "$1" "test -f \$HOME/$2"; then
    echo "Error: command failed or file not found: $1:$2"
    exit 1
  fi
}

echo "=== Checking data on $source_host ==="
ensure_dir $source_host data
ensure_file $source_host data/setup.txt
ensure_dir $source_host grafana
ensure_dir $source_host influxdb
echo

echo "=== Checking data on $target_host ==="
ensure_dir $target_host data
ensure_dir $target_host grafana
ensure_dir $target_host influxdb
echo

echo "=== Creating LODA checkpoint on $source_host ==="
gssh $source_host "curl -X POST localhost/miner/v1/checkpoint"
ensure_file $source_host data/checkpoint.txt
echo

echo "=== Fetching LODA checkpoint and setup from $source_host ==="
gscp "$source_host:data/checkpoint.txt" .
gscp "$source_host:data/setup.txt" .
echo

echo "=== Stopping LODA services on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/supervisorctl stop oeis programs stats"
echo

echo "=== Pushing LODA checkpoint and setup to $target_host ==="
gscp checkpoint.txt $target_host:data/
gscp setup.txt $target_host:data/
ensure_file $target_host data/checkpoint.txt
ensure_file $target_host data/setup.txt
rm_local checkpoint.txt setup.txt
echo

echo "=== Starting LODA services on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/supervisorctl start oeis programs stats"
echo

echo "=== Creating influxdb-backup on $source_host ==="
gssh $source_host "docker exec loda-api /usr/bin/influxd backup -portable /influxdb-backup"
gssh $source_host "docker cp loda-api:/influxdb-backup ."
gssh $source_host "docker exec loda-api rm -R /influxdb-backup"
ensure_dir $source_host influxdb-backup
echo

echo "=== Fetching influxdb-backup from $source_host ==="
gscp "$source_host:influxdb-backup" .
gssh $source_host "rm -R influxdb-backup"
echo

echo "=== Pushing influxdb-backup to $target_host ==="
gscp influxdb-backup $target_host
gssh $target_host "docker cp influxdb-backup loda-api:/"
gssh $target_host "rm -R influxdb-backup"
rm_local influxdb-backup
echo

echo "=== Restoring influxdb-backup on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/influxd restore -portable /influxdb-backup"
gssh $target_host "docker exec loda-api rm -R /influxdb-backup"
echo

echo "=== Fetching grafana data from $source_host ==="
gscp "$source_host:grafana" .
echo

echo "=== Stopping grafana on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/supervisorctl stop grafana"
echo

echo "=== Pushing grafana data to $target_host ==="
gscp grafana $target_host
echo

echo "=== Starting grafana on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/supervisorctl start grafana"
rm_local grafana
echo

echo "=== Starting all services on $target_host ==="
gssh $target_host "docker exec loda-api /usr/bin/supervisorctl status"
echo

echo "=== Finished ==="
