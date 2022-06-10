# LODA API Server and Monitoring

This project consists of the following parts:

* LODA API server implementations written in Go.
* Monitoring of LODA miners using InfluxDB and Grafana.
* Docker image for running everything in one container.

There are separate API servers for:

* OEIS cache
* Program submissions
* Stats

The Docker image is based on the images by [Phil Hawthorne](https://github.com/philhawthorne/docker-influxdb-grafana) and [Samuele Bistoletti](https://github.com/samuelebistoletti/docker-statsd-influxdb-grafana).
