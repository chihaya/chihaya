.DEFAULT_GOAL := build
.PHONY: *

VERBOSE ?=
GO_FLAGS := $(if $(VERBOSE),-v,)

clean:
	rm -rf ./bin
 
build:
	go build $(GO_FLAGS) -o "bin/chihaya" ./cmd/...

build-container:
	docker build -t chihaya:dev .

bench:
	go test $(GO_FLAGS) ./... -bench=. -benchtime=5s
 
test: test-unit test-e2e test-helm

test-unit:
	go test $(GO_FLAGS) -failfast -race -count=1 -timeout=5m ./...

test-e2e: test-e2e-mem test-e2e-redis

test-e2e-mem:
	@tmpdir=$$(mktemp -d); \
	trap 'kill $$pid 2>/dev/null; rm -rf "$$tmpdir"' EXIT; \
	go build -o "$$tmpdir/chihaya" ./cmd/chihaya; \
	$$tmpdir/chihaya --config=./dist/example_config.yaml --debug & \
	pid=$$!; \
	sleep 2; \
	$$tmpdir/chihaya e2e --debug

test-e2e-redis:
	@tmpdir=$$(mktemp -d); \
	trap 'kill $$rpid 2>/dev/null; kill $$cpid 2>/dev/null; rm -rf "$$tmpdir"; sleep 2' EXIT; \
	go build -o "$$tmpdir/chihaya" ./cmd/chihaya; \
	redis-server --save "" & \
	rpid=$$!; \
	$$tmpdir/chihaya --config="./dist/example_config_redis.yaml" --debug & \
	cpid=$$!; \
	sleep 2; \
	$$tmpdir/chihaya e2e --debug

test-helm:
	cd ./dist/helm/chihaya && helm template . --debug

lint: lint-go lint-yaml

lint-go: lint-gofmt lint-gomod lint-golangci

lint-gofmt:
	go tool mvdan.cc/gofumpt -d .

lint-gomod:
	go mod tidy -diff

lint-golangci:
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./...

lint-yaml:
	go tool github.com/google/yamlfmt/cmd/yamlfmt -lint .

fix-fmt:
	go tool mvdan.cc/gofumpt -w .

fix-gomod:
	go mod tidy
