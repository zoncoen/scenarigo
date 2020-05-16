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

```yaml example.yaml
title: check scenarigo GitHub page
steps:
- title: GET https://api.github.com/repos/zoncoen/scenarigo
  vars:
    user: zoncoen
    repo: scenarigo
  protocol: http
  request:
    method: GET
    url: "https://api.github.com/repos/{{vars.user}}/{{vars.repo}}"
  expect:
    code: OK
    body:
      name: "{{vars.repo}}"
```

```shell
$ scenarigo run example.yaml
ok      example.yaml    0.000s
```

## Features

* provides the command-line tool and the Go package for testing
* supports HTTP and gRPC
* customization by writing Go code

## Installation

Go to the [releases page](https://github.com/zoncoen/scenarigo/releases) and download the zip file. Unpack the zip file, and put the binary to a directory in your `$PATH`.

## Usage

```shell
scenarigo is a scenario testing tool for APIs.

Usage:
  scenarigo [command]

Available Commands:
  help        Help about any command
  run         run test scenarios
  version     print scenarigo version

Flags:
  -h, --help   help for scenarigo

Use "scenarigo [command] --help" for more information about a command.
```

## Development

This project uses the Makefile as a task runner.

### Available commands

```shell
test                           run tests
coverage                       measure test coverage
gen                            generate necessary files for testing
release                        release new version
changelog                      generate CHANGELOG.md
credits                        generate CREDITS
help                           print help
```
