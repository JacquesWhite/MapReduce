cp ./../proto/master.proto ./master.proto

python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. master.proto