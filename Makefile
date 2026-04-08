BINARY    := go-grpc-pong
PROTO_DIR := proto
PB_DIR    := pb

.PHONY: all build proto lint test clean

all: proto build

proto:
	protoc \
		--go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) $(PROTO_DIR)/pong.proto

build:
	go build -o $(BINARY) .

lint:
	go vet ./...
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, running go vet only. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

test:
	go test -v -race ./...

clean:
	rm -f $(BINARY) $(PB_DIR)/*.go
