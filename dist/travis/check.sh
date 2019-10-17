#!/bin/bash
set -e

go test -v -race $(go list ./...)
go vet $(go list ./...)
diff <(goimports -local github.com/chihaya/chihaya -d $(find . -type f -name '*.go' -not -path "./vendor/*")) <(printf "")
(for d in $(go list ./...); do diff <(golint $d) <(printf "") || exit 1;  done)
go install github.com/chihaya/chihaya/cmd/chihaya

# Run e2e test with example config.
chihaya --config=./dist/travis/config_memory.yaml --debug&
pid=$!
sleep 2 # wait for Chihaya to start up (gross)
chihaya e2e --debug
kill $pid

# Run e2e test with redis.
chihaya --config=./dist/travis/config_redis.yaml --debug&
pid=$!
sleep 2 # wait for Chihaya to start up (gross)
chihaya e2e --debug
kill $pid