#!/bin/bash
# Script that builds plugin,
# pass it to the Worker alongside other arguments
# and run the worker

# Build plugin for Worker
(cd plugins && go build -buildmode=plugin worker-plugin.go) || exit 1

# Build Worker
(cd main && go build main-worker.go) || exit 1


# Run Mock Master-Server
#PORT=$RANDOM
#(cd main && go build mock-master.go) || exit 1
#timeout -k 2s 180s ./main/mock-master $PORT ../datasets/test.txt
#sleep 1

# Run Worker
timeout -k 2s 180s ./main/main-worker 127.0.0.1 12345 plugins/worker-plugin.so