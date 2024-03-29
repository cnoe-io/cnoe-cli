#!/bin/bash

set -e -x -u

go mod tidy
# go test ./...
go fmt ./cmd/... ./pkg/...

# build without website assets
go build -o cnoe ./cmd/...
go build -tags embed -o cnoe-embed ./cmd/...
./cnoe version
