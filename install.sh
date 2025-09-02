#!/bin/bash

set -euo pipefail

function checkpasswd {
  if [ -z "$1" ]; then
    echo "Password not set"
    exit 1
  fi
}

# ask for initial influx passwords
if [ ! -d "$HOME/influxdb" ]; then
  echo -n "Set InfluxDB root password: "
  read INFLUXDB_ROOT_PASSWD
  checkpasswd $INFLUXDB_ROOT_PASSWD
  echo -n "Set InfluxDB loda password: "
  read INFLUXDB_LODA_PASSWD
  checkpasswd $INFLUXDB_LODA_PASSWD
  mkdir $HOME/influxdb
fi

# ask for initial grafana passpord
if [ ! -d "$HOME/grafana" ]; then
  echo -n "Set Grafana root password: "
  read GRAFANA_ROOT_PASSWD
  checkpasswd $GRAFANA_ROOT_PASSWD
  mkdir -p $HOME/grafana/dashboards
  cp ./image/home.json $HOME/grafana/dashboards/home.json
fi

# create data directory
if [ ! -d "$HOME/data" ]; then
  mkdir -p $HOME/data
fi
if [ ! -f "$HOME/data/setup.txt" ]; then
  echo "LODA_INFLUXDB_AUTH=loda:loda@$INFLUXDB_LODA_PASSWD" > "$HOME/data/setup.txt"
  echo "LODA_INFLUXDB_HOST=http://localhost/influxdb" >> "$HOME/data/setup.txt"
  echo "LODA_LOG_DIR=/var/log/loda" >> "$HOME/data/setup.txt"
fi
if [ ! -f "$HOME/data/openapi.v2.yaml" ]; then
  cp ./openapi.v2.yaml $HOME/data/openapi.v2.yaml
fi

echo
echo "### BUILDING LODA IMAGE ###"
docker build -t loda-api .

# check if there is already a running container
CONT=$(docker ps -q -f name=loda-api)
if [ -n "$CONT" ]; then
  echo "There is already a running loda-api container. Stop and replace it? (yes/no)"
  read ANSWER
  if [ "$ANSWER" != "yes" ]; then
    echo "Aborting."
    exit 1
  fi
  docker stop $CONT
fi

echo
echo "### START CONTAINER ###"
docker run -d --rm --name loda-api --hostname lodaapi -p 80:80 -p 443:443 -v $HOME/influxdb:/var/lib/influxdb -v $HOME/grafana:/var/lib/grafana -v $HOME/data:/data loda-api:latest

if [[ -v INFLUXDB_ROOT_PASSWD ]]; then
  echo
  echo "### CONFIGURE INFLUXDB ###"
  sleep 5
  docker exec loda-api influx -execute "CREATE USER root WITH PASSWORD '$INFLUXDB_ROOT_PASSWD' WITH ALL PRIVILEGES"
  docker exec loda-api influx -username=root -password=$INFLUXDB_ROOT_PASSWD -execute "CREATE USER loda WITH PASSWORD '$INFLUXDB_LODA_PASSWD'"
  docker exec loda-api influx -username=root -password=$INFLUXDB_ROOT_PASSWD -execute "CREATE DATABASE loda WITH DURATION 14d"
  docker exec loda-api influx -username=root -password=$INFLUXDB_ROOT_PASSWD -execute "GRANT ALL ON loda TO loda"
fi

if [[ -v GRAFANA_ROOT_PASSWD ]]; then
  echo
  echo "### CONFIGURE GRAFANA ###"
  sleep 5
  docker exec loda-api grafana-cli admin reset-admin-password $GRAFANA_ROOT_PASSWD
  cat ./image/datasource.json | sed "s/INFLUXDB_LODA_PASSWD/$INFLUXDB_LODA_PASSWD/" > /tmp/ds.json
  curl -i -u root:$GRAFANA_ROOT_PASSWD -X POST -H "Content-Type: application/json" -d @/tmp/ds.json http://localhost/grafana/api/datasources
  rm /tmp/ds.json
fi

echo
echo "### HEALTH CHECKS ###"
sleep 10
curl -sS "http://localhost/miner/v1/count"
echo
curl -sS "http://localhost/v2/openapi"
echo
curl -sS "http://localhost/grafana/api/health"
echo
curl -sS "http://localhost/influxdb/health"

echo
echo "### FINISHED ###"
