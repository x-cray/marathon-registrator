# Marathon Service Registrator
[![Build Status](https://travis-ci.org/x-cray/marathon-registrator.svg?branch=master)](https://travis-ci.org/x-cray/marathon-registrator)
[![Docker Pulls](https://img.shields.io/docker/pulls/xcray/marathon-registrator.svg)](https://hub.docker.com/r/xcray/marathon-registrator/)
[![](https://badge.imagelayers.io/xcray/marathon-registrator:latest.svg)](https://imagelayers.io/?images=xcray/marathon-registrator:latest 'Get your own badge on imagelayers.io')

Consul service registry bridge for Marathon. It monitors services running by Marathon and syncs them to Consul.
Heavily inspired by [registrator](https://github.com/gliderlabs/registrator).

# Features
* Automatically registers/deregisters Marathon tasks as services in Consul.
* Real-time updates of service registry.
* Uses new Marathon [Event Stream](https://mesosphere.github.io/marathon/docs/rest-api.html#event-stream)
(e.g. /v2/events) for getting service updates. No need to reconfigure Marathon to use webhooks.
* Automatic cleanup of dangling services from service registry.
* Designed with extensibility in mind: service scheduler and service registry are
abstractions which may have different implementations. Currently, there is only Marathon
scheduler and Consul service registry implemented.

# Installation

# Usage

## Options
|       Option      | Description |
| ----------------- |------------ |
| `consul`          | Address and port of Consul agent. Default: `http://127.0.0.1:8500`.
| `marathon`        | URL of Marathon instance. Multiple instances may be specified in case of HA setup: http://addr1:8080,addr2:8080,addr3:8080. Default: `http://127.0.0.1:8080`.
| `resync-interval` | Time interval to resync Marathon services to determine dangling instances. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Default: `5m`.
| `dry-run`         | Do not perform actual service registeration/deregistration. Just log intents.
| `log-level`       | Set the logging level - valid values are "debug", "info", "warn", "error", and "fatal". Default: `info`.
| `force-colors`    | Force colored log output.

# Development
Install dependencies:
```shell
$ make deps
```

Run tests:
```shell
$ make test
```

Generate mocks for interfaces (run when you modify mocked type interface):
```shell
$ make mocks
```

# License
MIT
