#!/bin/bash

# Exit on any error
set -e

# Build the binary
cd ../cmd/gocache
go build -o gocache

# Start three cache servers
./gocache --port=8001 --log=debug &
PID1=$!
sleep 1

./gocache --port=8002 --log=debug &
PID2=$!
sleep 1

./gocache --port=8003 --log=debug &
PID3=$!
sleep 1

# Start an API server
./gocache --port=8004 --api --log=debug &
PID4=$!
sleep 2

# Test the cache
echo "Testing cache servers..."

# Test direct cache access
echo "Testing direct cache access..."
curl -s "http://localhost:8001/_gocache/scores/Tom" > /dev/null
if [ $? -eq 0 ]; then
  echo "✅ Direct cache access successful"
else
  echo "❌ Direct cache access failed"
fi

# Test API server
echo "Testing API server..."
curl -s "http://localhost:9999/api/cache?group=scores&key=Tom" > /dev/null
if [ $? -eq 0 ]; then
  echo "✅ API server access successful"
else
  echo "❌ API server access failed"
fi

# Test health endpoint
echo "Testing health endpoint..."
curl -s "http://localhost:9999/health" > /dev/null
if [ $? -eq 0 ]; then
  echo "✅ Health endpoint successful"
else
  echo "❌ Health endpoint failed"
fi

# Clean up
echo "Cleaning up..."
kill $PID1 $PID2 $PID3 $PID4

echo "All tests completed!" 