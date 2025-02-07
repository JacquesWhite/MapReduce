#!/bin/bash

# Store the current directory
current_dir=$(pwd)

# Change to the directory where the script is located
cd "$(dirname "$0")" || exit

cp ./../proto/master.proto ./master.proto

python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. master.proto

# Change back to the original directory
cd ${current_dir} || exit