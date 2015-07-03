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
ADD cmd        /go/src/github.com/chihaya/chihaya/cmd
ADD config     /go/src/github.com/chihaya/chihaya/config
ADD deltastore /go/src/github.com/chihaya/chihaya/deltastore
ADD stats      /go/src/github.com/chihaya/chihaya/stats
ADD store      /go/src/github.com/chihaya/chihaya/store
ADD tracker    /go/src/github.com/chihaya/chihaya/tracker
ADD transport  /go/src/github.com/chihaya/chihaya/transport

# Install
RUN go install github.com/chihaya/chihaya/cmd/chihaya

# docker run -p 6881:6881 -v $PATH_TO_DIR_WITH_CONF_FILE:/config quay.io/jzelinskie/chihaya
VOLUME ["/config"]
EXPOSE 6881
CMD ["chihaya", "-config=/config/config.json", "-logtostderr=true"]
