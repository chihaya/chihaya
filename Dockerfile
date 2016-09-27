# vim: ft=dockerfile
FROM golang:alpine
MAINTAINER Jimmy Zelinskie <jimmyzelinskie@gmail.com>

# Create source directory
WORKDIR        /go/src/github.com/chihaya/chihaya/
RUN mkdir -p   /go/src/github.com/chihaya/chihaya/

# Install dependencies
RUN apk update && apk add git
RUN go get github.com/tools/godep
ADD Godeps /go/src/github.com/chihaya/chihaya/Godeps
RUN godep restore

# Add source files
ADD *.go       /go/src/github.com/chihaya/chihaya/
ADD api        /go/src/github.com/chihaya/chihaya/api
ADD cmd        /go/src/github.com/chihaya/chihaya/cmd
ADD config     /go/src/github.com/chihaya/chihaya/config
ADD http       /go/src/github.com/chihaya/chihaya/http
ADD stats      /go/src/github.com/chihaya/chihaya/stats
ADD tracker    /go/src/github.com/chihaya/chihaya/tracker
ADD udp        /go/src/github.com/chihaya/chihaya/udp
ADD example_config.json /config.json

# Install chihaya
RUN go install github.com/chihaya/chihaya/cmd/chihaya

# Setup the entrypoint
# docker run -p 6880-6882:6880-6882 -v $PATH_TO_CONFIG_FILE:/config.json:ro quay.io/jzelinskie/chihaya:latest -v=5
EXPOSE 6880-6882
ENTRYPOINT ["chihaya", "-config=/config.json", "-logtostderr=true"]
CMD ["-v=5"]
