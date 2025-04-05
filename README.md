# Go-Cache

A distributed caching system written in Go, designed to be both simple to use and highly efficient.

## Features

- LRU cache eviction
- Distributed, horizontally scalable architecture
- Consistent hashing for peer selection
- Protobuf-based communication for efficiency
- HTTP API for easy integration
- Configurable via file, environment variables, or command-line flags
- Logging with multiple output formats
- Thread-safe operations

## Installation

```bash
go get github.com/AdrianWangs/go-cache-new
```

## Quick Start

1. Build the binary:

```bash
cd cmd/gocache
go build -o gocache
```

2. Run a single cache server:

```bash
./gocache --port=8001
```

3. Run multiple servers to create a cluster:

```bash
./gocache --port=8001 &
./gocache --port=8002 &
./gocache --port=8003 &
```

4. Start with an API server:

```bash
./gocache --port=8001 --api
```

## API Usage

Get a value from cache:

```
GET /api/cache?group=scores&key=Tom
```

Health check:

```
GET /health
```

## Configuration

Go-Cache can be configured through:

1. Command-line flags:

   - `--port`: Port to run the cache server on
   - `--api`: Start API server
   - `--log`: Log level (debug, info, warn, error)
   - `--config`: Path to configuration file

2. Environment variables:

   - `GOCACHE_MAX_BYTES`: Maximum cache size in bytes
   - `GOCACHE_PORT`: Port to run the cache server on
   - `GOCACHE_API_PORT`: Port to run the API server on
   - `GOCACHE_HOST`: Host to bind to
   - `GOCACHE_PEERS`: Comma-separated list of peer addresses
   - `GOCACHE_LOG_LEVEL`: Log level
   - `GOCACHE_LOG_FORMAT`: Log format (text or json)

3. Configuration file (JSON):

```json
{
  "max_cache_bytes": 104857600,
  "default_cache_expiry_seconds": 3600,
  "api_port": 9999,
  "cache_port": 8001,
  "host": "localhost",
  "base_path": "/_gocache/",
  "peer_addresses": ["http://localhost:8001", "http://localhost:8002", "http://localhost:8003"],
  "log_level": "info",
  "log_format": "text"
}
```

## Architecture

Go-Cache is designed with a clean architecture:

- `pkg/lru`: LRU cache implementation
- `pkg/logger`: Structured logging
- `internal/cache`: Core cache functionality
- `internal/consistenthash`: Consistent hashing algorithm
- `internal/peers`: Peer selection interfaces
- `internal/server`: HTTP server implementation
- `api`: API server
- `config`: Configuration management
- `cmd/gocache`: Command-line application

## License

MIT
