#!/bin/bash

# Store the current directory
current_dir=$(pwd)

# Change to the directory where the script is located
cd "$(dirname "$0")" || exit

generate_protocol_buffers() {
  mkdir -p ../proto/master
  mkdir -p ../proto/worker

  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

  protoc --proto_path=../proto --go_out=../proto/master --go_opt=paths=source_relative --go-grpc_out=../proto/master --go-grpc_opt=paths=source_relative ./../proto/master.proto
  protoc --proto_path=../proto --go_out=../proto/worker --go_opt=paths=source_relative --go-grpc_out=../proto/worker --go-grpc_opt=paths=source_relative ./../proto/worker.proto
}

build_master() {
  mkdir -p ../bin
  go build -o ../bin/master ../runner/master_runner.go
}

build_worker() {
  mkdir -p ../bin
  go build -o ../bin/worker ../runner/worker_runner.go
}

build_fake_worker() {
  mkdir -p ../bin
  go build -o ../bin/fake_worker ../runner/fake_worker_runner.go
}

build_worker_plugin() {
  mkdir -p ../bin
  go build -buildmode=plugin -o ../bin/worker_plugin.so ../worker/plugins/worker-plugin.go
}

echo "Building the project..."
echo "Generating protocol buffers..."
generate_protocol_buffers

# Install dependencies
echo "Installing dependencies..."
go mod tidy

# Build the master
echo "Building the master..."
build_master
echo "Building the worker..."
build_worker
echo "Building the fake worker..."
build_fake_worker
echo "Building the worker plugin..."
build_worker_plugin

echo "Finished"
# Change back to the original directory
cd ${current_dir} || exit
