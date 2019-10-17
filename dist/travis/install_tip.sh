#!/bin/bash
set -e

# Install golint and go vet.
go get -u golang.org/x/lint/golint
go get -u golang.org/x/tools/cmd/...

go get -t -u ./...