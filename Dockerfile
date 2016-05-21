# vim: ft=dockerfile
FROM golang
MAINTAINER Jimmy Zelinskie <jimmyzelinskie@gmail.com>

# Install glide
WORKDIR /tmp
ADD https://github.com/Masterminds/glide/releases/download/0.10.2/glide-0.10.2-linux-amd64.tar.gz /tmp
RUN tar xvf /tmp/glide-0.10.2-linux-amd64.tar.gz
RUN mv /tmp/linux-amd64/glide /usr/bin/glide

# Add files
WORKDIR        /go/src/github.com/chihaya/chihaya/
RUN mkdir -p   /go/src/github.com/chihaya/chihaya/

# Add source
ADD . .

# Install chihaya
RUN glide install
RUN go install github.com/chihaya/chihaya/cmd/chihaya

# Configuration/environment
VOLUME ["/config"]
EXPOSE 6880-6882

# docker run -p 6880-6882:6880-6882 -v $PATH_TO_DIR_WITH_CONF_FILE:/config:ro -e quay.io/jzelinskie/chihaya:latest
ENTRYPOINT ["chihaya", "-config=/config/config.json"]
