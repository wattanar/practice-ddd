.PHONY: all build run test test-verbose clean lint fmt db-clean

APP     := server
CMD_DIR := ./cmd/server
DB      := orders.db

all: build

build:
	go build -o bin/$(APP) $(CMD_DIR)

run:
	go run $(CMD_DIR)

run-build: build
	./bin/$(APP)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -race ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/ $(DB)

db-clean:
	rm -f $(DB) $(DB)-shm $(DB)-wal
