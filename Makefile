PROTOFILES=ekiden/grpc/client.proto ekiden/grpc/enclaverpc.proto
GRPCFILES=$(patsubst %.proto,%.pb.go,$(PROTOFILES))

all:  build test build-gateway build-ekiden-client

build: build-grpc
	go build ./...

build-grpc: $(GRPCFILES)

%.pb.go: %.proto
	protoc -I ./ --go_out=plugins=grpc,paths=source_relative:. $<

build-gateway:
	go build -o developer-gateway github.com/oasislabs/developer-gateway/cmd/gateway

build-ekiden-client:
	go build -o ekiden-client github.com/oasislabs/developer-gateway/cmd/ekiden-client

lint:
	go vet ./...
	golangci-lint run

test:
	go test -v -race ./...

test-coverage:
	go test -v -covermode=count -coverprofile=coverage.out ./...

show-coverage:
	go tool cover -html=coverage.out

clean:
	rm -f developer-gateway
	go clean ./...
