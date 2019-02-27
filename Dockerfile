FROM golang:alpine AS build-env
LABEL maintainer "Jimmy Zelinskie <jimmyzelinskie+git@gmail.com>"

# Install OS-level dependencies.
RUN apk add --no-cache curl git

# Copy our source code into the container.
WORKDIR /go/src/github.com/ProtocolONE/chihaya
COPY . /go/src/github.com/ProtocolONE/chihaya

# Install our golang dependencies and compile our binary.
RUN CGO_ENABLED=0 GO111MODULE=on go install github.com/ProtocolONE/chihaya/cmd/...

FROM alpine:3.9
RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/bin/chihaya /chihaya

RUN adduser -D chihaya

# Expose a docker interface to our binary.
EXPOSE 6880 6969

# Drop root privileges
USER chihaya

ENTRYPOINT ["/chihaya"]
