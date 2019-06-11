PROTOFILES=ekiden/grpc/client.proto ekiden/grpc/enclaverpc.proto
GRPCFILES=$(patsubst %.proto,%.pb.go,$(PROTOFILES))

all:  build test build-cmd

build: build-grpc
	go build ./...

build-grpc: $(GRPCFILES)

%.pb.go: %.proto
	protoc -I ./ --go_out=plugins=grpc,paths=source_relative:. $<

build-cmd: build-gateway build-ekiden-client build-eth-client

build-gateway:
	go build -o developer-gateway github.com/oasislabs/developer-gateway/cmd/gateway

build-ekiden-client:
	go build -o ekiden-client github.com/oasislabs/developer-gateway/cmd/ekiden-client

build-eth-client:
	go build -o eth-client github.com/oasislabs/developer-gateway/cmd/eth-client

lint:
	go vet ./...
	golangci-lint run

test:
	OASIS_DG_CONFIG_PATH=config/dev.toml go test -v -race ./...

test-coverage:
	OASIS_DG_CONFIG_PATH=config/dev.toml go test -v -covermode=count -coverprofile=coverage.out ./...

test-lua:
	redis-cli --eval mqueue/redis/redis.lua , test

test-component:
	mkdir -p output
	go test -v -covermode=count -coverprofile=output/coverage.out github.com/oasislabs/developer-gateway/tests

test-component-redis-single:
	OASIS_DG_CONFIG_PATH=config/redis_single.toml go test -v -covermode=count -coverprofile=coverage.redis_single.out github.com/oasislabs/developer-gateway/tests

test-component-redis-cluster:
	OASIS_DG_CONFIG_PATH=config/redis_cluster.toml go test -v -covermode=count -coverprofile=coverage.redis_cluster.out github.com/oasislabs/developer-gateway/tests

test-component-dev:
	OASIS_DG_CONFIG_PATH=config/dev.toml go test -v -covermode=count -coverprofile=coverage.dev.out github.com/oasislabs/developer-gateway/tests

show-coverage:
	go tool cover -html=coverage.out

clean:
	rm -f developer-gateway
	rm -f ekiden-client
	rm -f eth-client
	rm -f $GRPCFILES
	rm -rf output
	go clean ./...
