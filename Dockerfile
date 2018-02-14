FROM golang:alpine AS build-env
LABEL maintainer "Jimmy Zelinskie <jimmyzelinskie+git@gmail.com>"

# Install OS-level dependencies.
RUN apk update && \
    apk add curl git

# Copy our source code into the container.
WORKDIR /go/src/github.com/chihaya/chihaya
COPY . /go/src/github.com/chihaya/chihaya

# Install our golang dependencies and compile our binary.
RUN go get -u github.com/golang/dep/...
RUN dep ensure
RUN CGO_ENABLED=0 go install github.com/chihaya/chihaya/cmd/...

FROM alpine:latest
COPY --from=build-env /go/bin/chihaya /chihaya

RUN adduser -D chihaya

# Expose a docker interface to our binary.
EXPOSE 6880 6969

# Drop root privileges
USER chihaya

ENTRYPOINT ["/chihaya"]
