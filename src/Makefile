.PHONY: default buildProto buildWorker buildPluginWordcount buildPluginCrash runWorker runMaster build clean

default: build

buildMaster:
	mkdir -p ../bin
	go build -o bin/master runner/master_runner.go

buildWorker:
	mkdir -p ../bin
	go build -o bin/worker runner/worker_runner.go

buildPluginWordcount:
	mkdir -p ../bin
	go build -buildmode=plugin -o bin/worker_plugin.so worker/plugins/wordcount/worker-plugin-wordcount.go

buildPluginCrash:
	mkdir -p ../bin
	go build -buildmode=plugin -o bin/worker_plugin.so worker/plugins/crash/worker-plugin-crash.go

buildProto:
	mkdir -p proto/master
	mkdir -p proto/worker

	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

	protoc --proto_path=proto --go_out=proto/master --go_opt=paths=source_relative --go-grpc_out=proto/master --go-grpc_opt=paths=source_relative ./proto/master.proto
	protoc --proto_path=proto --go_out=proto/worker --go_opt=paths=source_relative --go-grpc_out=proto/worker --go-grpc_opt=paths=source_relative ./proto/worker.proto

runWorker:
	./bin/worker 127.0.0.1 12382 127.0.0.1 worker/plugins/worker-plugin.so

runMaster:
	./bin/master --port=12382

build: buildProto buildMaster buildWorker buildPluginWordcount

clean:
	rm -r bin
	rm -rf intermediate
	rm -f output