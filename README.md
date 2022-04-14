# Prometheus Powerpal Exporter

This exporter is built to expose the live data from a [Powerpal](http://powerpal.net) power 
consumption monitoring device into a format that can be pulled into Prometheus

[![CircleCI](https://circleci.com/gh/aashley/powerpal_exporter/tree/main.svg?style=svg)](https://circleci.com/gh/aashley/powerpal_exporter/tree/main)

# Usage

## Installation

Binaries can be downloaded from the [Github 
releases](https://github.com/aashley/powerpal_exporter/releases) and need no
special installation.

## Running

Start `powerpal_exporter` as a daemon or from CLI specifying your API token and device ID:

```sh
./powerpal_exporter --token "API_TOKEN" --device "DEVICE_ID" --refresh 30
```

Visit http://localhost:9915/metrics to get all the metrics exposed.

Command line parameters can be set using environment variables instead.

| Option    | Default | Environment Variable |
|-----------|---------|----------------------|
| --token   |         | POWERPAL_TOKEN       |
| --device  |         | POWERPAL_DEVICE      |
| --refresh | 30      | POWERPAL_REFRESH     |

## Docker

Powerpal Exporter is available as a Docker image at https://hub.docker.com/r/adamashley/powerpal-exporter

The Docker image works with the same environment variables used by the daemon itself.