FROM golang:alpine
MAINTAINER Jimmy Zelinskie <jimmyzelinskie@gmail.com>

RUN apk update && apk add curl git
RUN curl https://glide.sh/get | sh

WORKDIR /go/src/github.com/chihaya/chihaya
ADD . /go/src/github.com/chihaya/chihaya
RUN glide install
RUN go install github.com/chihaya/chihaya/cmd/chihaya

EXPOSE 6880 6881
ENTRYPOINT ["chihaya"]
