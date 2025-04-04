#!/bin/bash

# This script tests cache performance with concurrent requests

# Exit on any error
set -e

# Build the binary
cd ../cmd/gocache
go build -o gocache

# Start the cache server
./gocache --port=8001 --log=info &
PID=$!
sleep 2

# Run concurrent requests
echo "Running concurrent requests..."
for i in {1..100}; do
  curl -s "http://localhost:8001/_gocache/scores/Tom" > /dev/null &
done

# Wait for all requests to complete
wait

# Clean up
echo "Cleaning up..."
kill $PID

echo "Concurrent test completed!" 