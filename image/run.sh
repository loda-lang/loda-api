#!/bin/bash -e

# We need to ensure this directory is writeable on start of the container
chmod 0777 /var/lib/grafana

# Initialize Perses datasource and home dashboard if not already done
if [ ! -f /var/lib/perses/datasources/InfluxDB.json ]; then
    mkdir -p /var/lib/perses/datasources
    if [ -f /root/perses-datasource.json ]; then
        cp /root/perses-datasource.json /var/lib/perses/datasources/InfluxDB.json
    else
        echo "Warning: /root/perses-datasource.json not found, Perses datasource not configured"
    fi
fi

if [ ! -f /var/lib/perses/dashboards/home.json ]; then
    mkdir -p /var/lib/perses/dashboards
    if [ -f /root/perses-home.json ]; then
        cp /root/perses-home.json /var/lib/perses/dashboards/home.json
    else
        echo "Warning: /root/perses-home.json not found, Perses home dashboard not configured"
    fi
fi

exec /usr/bin/supervisord
