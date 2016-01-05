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

# docker run -p 6881:6881 -v $PATH_TO_DIR_WITH_CONF_FILE:/config quay.io/jzelinskie/chihaya
VOLUME ["/config"]
EXPOSE 6881
CMD ["chihaya", "-config=/config/config.json", "-logtostderr=true"]
