#cd ..
protoc --go_out=src/ --go-grpc_out=src/ src/proto/master.proto
protoc --go_out=src/ --go-grpc_out=src/ src/proto/worker.proto

#mv generated_files/github.com/Jacques generated_files/proto