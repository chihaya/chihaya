FROM golang:alpine AS build-env
LABEL maintainer "Jimmy Zelinskie <jimmyzelinskie@gmail.com>"

# Install OS-level dependencies.
RUN apk update && \
    apk add curl git && \
    curl https://glide.sh/get | sh

# Copy our source code into the container.
WORKDIR /go/src/github.com/chihaya/chihaya
COPY . /go/src/github.com/chihaya/chihaya

# Install our golang dependencies and compile our binary.
RUN glide install
RUN CGO_ENABLED=0 go install github.com/chihaya/chihaya/cmd/chihaya

FROM alpine:latest
COPY --from=build-env /go/bin/chihaya /chihaya

RUN adduser -D chihaya

# Expose a docker interface to our binary.
EXPOSE 6880 6881

# Drop root privileges
USER chihaya

ENTRYPOINT ["/chihaya"]
