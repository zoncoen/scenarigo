# scenarigo

[![godoc](https://godoc.org/github.com/zoncoen/scenarigo?status.svg)](https://pkg.go.dev/github.com/zoncoen/scenarigo)
![test](https://github.com/zoncoen/scenarigo/workflows/test/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/zoncoen/scenarigo/branch/master/graph/badge.svg)](https://codecov.io/gh/zoncoen/scenarigo)
[![go report](https://goreportcard.com/badge/zoncoen/scenarigo)](https://goreportcard.com/report/github.com/zoncoen/scenarigo)
[![codebeat](https://codebeat.co/badges/93ee2453-1a25-4db6-b98e-c430c994b4b8)](https://codebeat.co/projects/github-com-zoncoen-scenarigo-master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

An end-to-end scenario testing tool for HTTP/gRPC server.

## Overview

scenarigo is an end-to-end scenario testing tool for HTTP/gRPC server.
It is written in Go, enable to customize by [the plugin package of Go](https://golang.org/pkg/plugin/).
You can write test scenarios as YAML files and executes them.

## Development

### Testing

```shell
$ make gen
$ make test
```
