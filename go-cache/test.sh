#!/bin/bash

./main -port=8001 &
./main -port=8002 &
./main -port=8003 -api=1 &
