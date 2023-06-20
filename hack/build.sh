#!/bin/bash

set -e -x -u

# go test ./...
go fmt ./cmd/... ./pkg/...

# build without website assets
go build -o cnoe ./cmd/...
./cnoe version
