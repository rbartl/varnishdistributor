#!/bin/bash

docker build -t varnishdistributor-build .

docker run --rm -e GOOS=linux -v $(pwd)/out:/out -e CGO_ENABLED=0  varnishdistributor-build /usr/local/go/bin/go build -o /out/varnishdistributor

