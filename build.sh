#!/bin/bash

docker build -t varnishdistributor .

docker run --rm -e GOOS=linux -v $(pwd)/out:/out -e CGO_ENABLED=0  vo-schedules-build /usr/local/go/bin/go build -o /out/varnishdistributor

#docker run -it vo-schedules-build /usr/local/go/bin/go build
