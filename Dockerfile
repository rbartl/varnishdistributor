#FROM golang:onbuild
FROM iron/go:dev


RUN mkdir /app 
ADD . /app/ 
WORKDIR /app 
#RUN dep ensure

ENV SRC_DIR=/go/src/github.com/rbartl/varnishdistributor
# Add the source code:
ADD . $SRC_DIR

WORKDIR $SRC_DIR
#RUN cd $SRC_DIR; go build 

#CMD ["/usr/local/go/bin/go"]

