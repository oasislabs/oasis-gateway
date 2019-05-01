all: build test

build:
	go build github.com/oasislabs/developer-gateway/api
	go build github.com/oasislabs/developer-gateway/log
	go build github.com/oasislabs/developer-gateway/rpc

lint:
	go vet github.com/oasislabs/developer-gateway/api
	go vet github.com/oasislabs/developer-gateway/log
	go vet github.com/oasislabs/developer-gateway/rpc
	golangci-lint run

test:
	go test -v -race github.com/oasislabs/developer-gateway/api
	go test -v -race github.com/oasislabs/developer-gateway/log
	go test -v -race github.com/oasislabs/developer-gateway/rpc

test-coverage:
	go test -v -covermode=count -coverprofile=coverage.api.out github.com/oasislabs/developer-gateway/api
	go test -v -covermode=count -coverprofile=coverage.log.out github.com/oasislabs/developer-gateway/log
	go test -v -covermode=count -coverprofile=coverage.rpc.out github.com/oasislabs/developer-gateway/rpc
