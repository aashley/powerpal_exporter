# Prometheus Powerpal Exporter

This exporter is built to expose the live data from a [Powerpal](http://powerpal.net) power 
consumption monitoring device into a format that can be pulled into Prometheus

# Usage

## Installation

Binaries can be downloaded from the [Github 
releases](https://github.com/aashley/powerpal_exporter/releases) and need no
special installation.

## Running

Start `powerpal_exporter` as a daemon or from CLI specifying your API token and device ID:

```sh
./powerpal_exporter --token "API_TOKEN" --device "DEVICE_ID"
```

Visit http://localhost:9915/metrics to get all the metrics exposed.

