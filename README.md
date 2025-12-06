# LODA API Server and Monitoring

This project consists of the following parts:

* LODA API server implementations written in Go.
* Monitoring of LODA miners using InfluxDB, Grafana, and Perses.
* Docker image for running everything in one container.

There are separate API servers for:

* OEIS cache
* Program submissions
* Stats

## Dashboards

The monitoring dashboards are available at:

* Grafana: `https://dashboard.loda-lang.org/grafana/`
* Perses: `https://dashboard.loda-lang.org/perses/`

Both dashboards connect to the same InfluxDB data source and display LODA mining metrics.

## Recommended Hardware Setup

We recommend running this on a micro-instance with 1 GB memory, 1 shared CPU,
30 GB of standard persistent disk with Container-optimized OS.
