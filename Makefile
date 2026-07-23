.PHONY: all build test vet fmt check clean install

BINARY_NAME=minigit

all: check build

build:
	go build -o $(BINARY_NAME) ./cmd/minigit

test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

check: fmt vet test

clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf bin/ dist/ coverage.out

install:
	go install ./cmd/minigit
