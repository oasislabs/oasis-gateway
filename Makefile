all:  build test build-cmd

build: build-grpc
	go build ./...

build-grpc: ekiden/grpc/*.proto
	protoc -I ./ --go_out=plugins=grpc,paths=source_relative:. ekiden/grpc/*.proto

build-cmd: build-gateway build-ekiden-client build-eth-client

build-gateway:
	go build -o oasis-gateway github.com/oasislabs/oasis-gateway/cmd/gateway

build-ekiden-client:
	go build -o ekiden-client github.com/oasislabs/oasis-gateway/cmd/ekiden-client

build-eth-client:
	go build -o eth-client github.com/oasislabs/oasis-gateway/cmd/eth-client

lint:
	go vet ./...
	golangci-lint run

test:
	go test -v -race ./...

test-coverage:
	OASIS_DG_CONFIG_PATH=config/dev.toml go test -v -covermode=count -coverprofile=coverage.out ./...

test-lua:
	redis-cli --eval mqueue/redis/redis.lua , test

test-component:
	mkdir -p output
	go test -v -covermode=count -coverprofile=output/coverage.out github.com/oasislabs/oasis-gateway/tests

test-component-redis-single:
	OASIS_DG_CONFIG_PATH=config/redis_single.toml go test -v -covermode=count -coverprofile=coverage.redis_single.out github.com/oasislabs/oasis-gateway/tests

test-component-redis-cluster:
	OASIS_DG_CONFIG_PATH=config/redis_cluster.toml go test -v -covermode=count -coverprofile=coverage.redis_cluster.out github.com/oasislabs/oasis-gateway/tests

test-component-dev:
	OASIS_DG_CONFIG_PATH=config/dev.toml go test -v -covermode=count -coverprofile=coverage.dev.out github.com/oasislabs/oasis-gateway/tests

show-coverage:
	go tool cover -html=coverage.out

clean:
	rm -f oasis-gateway
	rm -f ekiden-client
	rm -f eth-client
	rm -f $(GRPCFILES)
	rm -rf output
