# Marathon Service Registrator
[![Build Status](https://travis-ci.org/x-cray/marathon-registrator.svg?branch=master)](https://travis-ci.org/x-cray/marathon-registrator)
[![Docker Pulls](https://img.shields.io/docker/pulls/xcray/marathon-registrator.svg)](https://hub.docker.com/r/xcray/marathon-registrator/)
[![](https://badge.imagelayers.io/xcray/marathon-registrator:latest.svg)](https://imagelayers.io/?images=xcray/marathon-registrator:latest 'Get your own badge on imagelayers.io')

Consul service registry bridge for Marathon. It monitors services running by Marathon and syncs them to Consul. Heavily inspired by [registrator](https://github.com/gliderlabs/registrator).

# Features
* Automatically registers/deregisters Marathon tasks as services in Consul.
* Uses new Marathon [Event Stream](https://mesosphere.github.io/marathon/docs/rest-api.html#event-stream) (e.g. /v2/events) for getting service updates. No need to reconfigure Marathon to use webhooks.
* Automatic cleanup of dangling services from service registry.
* Designed with extensibility in mind: service scheduler and service registry are abstractions which may have different implementations. Currently, there is only Marathon scheduler and Consul service registry implemented.

# Usage

# Development
Install dependencies:
```shell
$ make deps
```

Run tests:
```shell
$ make test
```

Generate mocks for interfaces (run when you modify mocked object interface):
```shell
$ make mocks
```

# License
MIT
