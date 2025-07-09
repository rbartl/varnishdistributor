#!/bin/bash

podman build -t varnishdistributor-build .

podman run --rm -e GOOS=linux -v $(pwd)/out:/out -e CGO_ENABLED=0 varnishdistributor-build bash -c "go mod tidy && go mod download && go build -o /out/varnishdistributor ./vdistribute.go"

