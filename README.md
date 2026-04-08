# go-grpc-pong

A lightweight gRPC ping/pong tool for testing and validating cluster-to-cluster connectivity, particularly useful for Istio service mesh deployments.

## Overview

`go-grpc-pong` provides two simple daemons:
- **Pong Server**: Listens for gRPC Ping requests and responds with Pong messages
- **Ping Client**: Continuously sends Ping requests to a Pong server and measures round-trip time (RTT)

This tool is ideal for:
- Testing gRPC connectivity between Kubernetes clusters
- Validating Istio multi-cluster configurations
- Monitoring network latency in service mesh environments
- Debugging cross-cluster communication issues

📘 **For Istio users**: See [ISTIO.md](ISTIO.md) for detailed Gateway, VirtualService, and multi-cluster configuration examples.

## Features

- ✅ Simple gRPC-based ping/pong protocol
- ✅ Configurable ping interval
- ✅ Round-trip time (RTT) measurements
- ✅ Graceful shutdown handling
- ✅ gRPC reflection enabled for debugging
- ✅ Docker/container ready
- ✅ Minimal dependencies

## Prerequisites

- Go 1.26.2 or later
- Protocol Buffers compiler (`protoc`) for development
- Docker (optional, for containerized deployment)

## Installation

### From Source

```bash
git clone https://github.com/h2ik/go-grpc-pong.git
cd go-grpc-pong
make build
```

This will create the `go-grpc-pong` binary in the current directory.

### Using Docker

```bash
docker build -t go-grpc-pong .
```

## Usage

### Start the Pong Server

The pong server listens for incoming gRPC ping requests:

```bash
./go-grpc-pong pong --addr :50051
```

**Flags:**
- `--addr`: Address to listen on (default: `:50051`)

**Example output:**
```
2026/04/08 12:00:00 Pong daemon listening on [::]:50051
2026/04/08 12:00:01 Received ping: "ping"
2026/04/08 12:00:02 Received ping: "ping"
```

### Start the Ping Client

The ping client sends periodic ping requests to the pong server:

```bash
./go-grpc-pong ping --addr localhost:50051 --interval 1s
```

**Flags:**
- `--addr`: Address of the pong server (default: `localhost:50051`)
- `--interval`: Time between pings (default: `1s`)

**Example output:**
```
2026/04/08 12:00:00 Connecting to pong server at localhost:50051
2026/04/08 12:00:00 Ping daemon started — sending pings every 1s
2026/04/08 12:00:01 Pong received: "pong"  rtt=1.234ms
2026/04/08 12:00:02 Pong received: "pong"  rtt=1.156ms
```

### Docker Usage

Run the pong server:
```bash
docker run -p 50051:50051 go-grpc-pong pong --addr :50051
```

Run the ping client:
```bash
docker run go-grpc-pong ping --addr <pong-server-address>:50051
```

## Kubernetes Deployment

Example deployment for the pong server:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pong-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pong-server
  template:
    metadata:
      labels:
        app: pong-server
    spec:
      containers:
      - name: pong
        image: go-grpc-pong:latest
        args: ["pong", "--addr", ":50051"]
        ports:
        - containerPort: 50051
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: pong-service
spec:
  selector:
    app: pong-server
  ports:
  - port: 50051
    targetPort: 50051
    protocol: TCP
```

Example deployment for the ping client:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ping-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ping-client
  template:
    metadata:
      labels:
        app: ping-client
    spec:
      containers:
      - name: ping
        image: go-grpc-pong:latest
        args: ["ping", "--addr", "pong-service:50051", "--interval", "2s"]
```

## Development

### Project Structure

```
.
├── cmd/               # Command implementations
│   ├── ping.go       # Ping client daemon
│   └── pong.go       # Pong server daemon
├── pb/               # Generated protobuf code
│   ├── pong.pb.go
│   └── pong_grpc.pb.go
├── proto/            # Protocol buffer definitions
│   └── pong.proto
├── main.go           # CLI entry point
├── Dockerfile        # Container image definition
├── Makefile          # Build automation
└── go.mod            # Go module definition
```

### Building

```bash
# Build everything (regenerate proto + compile)
make all

# Regenerate protobuf code only
make proto

# Build binary only
make build

# Run linters
make lint

# Run tests
make test

# Clean build artifacts
make clean
```

### Regenerating Protocol Buffers

If you modify `proto/pong.proto`, regenerate the Go code:

```bash
make proto
```

This requires:
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` plugin
- `protoc-gen-go-grpc` plugin

Install the plugins with:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Protocol Definition

The service uses a simple gRPC protocol defined in `proto/pong.proto`:

```protobuf
service PongService {
  rpc Ping(PingRequest) returns (PongResponse);
}

message PingRequest {
  string message = 1;
  int64  timestamp = 2;
}

message PongResponse {
  string message   = 1;
  int64  timestamp = 2;
}
```

## Testing with grpcurl

You can also test the pong server using [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# List services (reflection must be enabled)
grpcurl -plaintext localhost:50051 list

# Call the Ping method
grpcurl -plaintext -d '{"message": "ping", "timestamp": 1234567890}' \
  localhost:50051 pong.PongService/Ping
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is available under the MIT License.
