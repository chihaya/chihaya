#!/bin/bash
set -ex

# Install golint and go vet.
go get -u golang.org/x/lint/golint
go get -u golang.org/x/tools/cmd/...

go mod download