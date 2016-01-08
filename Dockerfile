# vim: ft=dockerfile
FROM golang
MAINTAINER Jimmy Zelinskie <jimmyzelinskie@gmail.com>

# Add files
WORKDIR        /go/src/github.com/chihaya/chihaya/
RUN mkdir -p   /go/src/github.com/chihaya/chihaya/

# Dependencies
RUN go get github.com/tools/godep
ADD Godeps /go/src/github.com/chihaya/chihaya/Godeps
RUN godep restore

# Add source
ADD *.go       /go/src/github.com/chihaya/chihaya/
ADD api        /go/src/github.com/chihaya/chihaya/api
ADD cmd        /go/src/github.com/chihaya/chihaya/cmd
ADD config     /go/src/github.com/chihaya/chihaya/config
ADD http       /go/src/github.com/chihaya/chihaya/http
ADD stats      /go/src/github.com/chihaya/chihaya/stats
ADD tracker    /go/src/github.com/chihaya/chihaya/tracker
ADD udp        /go/src/github.com/chihaya/chihaya/udp

# Install
RUN go install github.com/chihaya/chihaya/cmd/chihaya

# Configuration/environment
VOLUME ["/config"]
EXPOSE 6880-6882

# docker run -p 6880-6882:6880-6882 -v $PATH_TO_DIR_WITH_CONF_FILE:/config:ro -e quay.io/jzelinskie/chihaya:latest -v=5
ENTRYPOINT ["chihaya", "-config=/config/config.json", "-logtostderr=true"]
CMD ["-v=5"]
