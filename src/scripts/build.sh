#!/bin/bash

# Store the current directory
current_dir=$(pwd)

# Change to the directory where the script is located
cd "$(dirname "$0")"

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

generate_protocol_buffers

# Install dependencies
go mod tidy

# Build the master
build_master

# Change back to the original directory
cd ${current_dir}