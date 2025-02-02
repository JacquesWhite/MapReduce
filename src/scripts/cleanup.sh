#!/bin/bash
# Cleanup script for all created files for MapReduce

# Store the current directory
current_dir=$(pwd)

# Change to the directory where the script is located
cd "$(dirname "$0")" || exit

cleanup_protocol_buffers() {
  rm -rf ../proto/master
  rm -rf ../proto/worker
}

cleanup_bin() {
  rm -rf ../bin
}

cleanup_work() {
  rm -rf ../../work
}

echo "Cleaning up the project..."
echo "Cleaning up protocol buffers..."
cleanup_protocol_buffers

echo "Cleaning up the bin directory..."
cleanup_bin

#echo "Cleaning up the work directory..."
#cleanup_work

echo "Cleanup finished"
# Change back to the original directory
cd ${current_dir} || exit