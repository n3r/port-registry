.PHONY: build test clean

build:
	go build -o bin/port-server ./cmd/server
	go build -o bin/portctl ./cmd/portctl

test:
	go test ./...

clean:
	rm -rf bin/
