all:  build test build-gateway

build:
	go build ./...

build-gateway:
	go build -o developer-gateway github.com/oasislabs/developer-gateway/cmd/gateway

lint:
	go vet ./...
	golangci-lint run

test:
	go test -v -race ./...

test-coverage:
	go test -v -covermode=count -coverprofile=coverage.out ./...
	cat coverage.out >> coverage.txt

show-coverage:
	go tool cover -html=coverage.out
