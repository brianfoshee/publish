#! /bin/sh

set -eu

APP_DIR="/go/src/github.com/${GITHUB_REPOSITORY}/"

mkdir -p ${APP_DIR} && cp -r ./ ${APP_DIR} && cd ${APP_DIR}

echo "Running go get"
go get ./...

echo "Testing Project"
go test ./...

echo "Building Project"
go build ./cmd/publish