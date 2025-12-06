#!/bin/bash -e

# We need to ensure this directory is writeable on start of the container
chmod 0777 /var/lib/grafana

# Initialize Perses datasource and home dashboard if not already done
if [ ! -f /var/lib/perses/datasources/InfluxDB.json ]; then
    mkdir -p /var/lib/perses/datasources
    cp /root/perses-datasource.json /var/lib/perses/datasources/InfluxDB.json
fi

if [ ! -f /var/lib/perses/dashboards/home.json ]; then
    mkdir -p /var/lib/perses/dashboards
    cp /root/perses-home.json /var/lib/perses/dashboards/home.json
fi

exec /usr/bin/supervisord
