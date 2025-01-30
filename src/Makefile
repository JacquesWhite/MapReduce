.PHONY: runWorker runMaster build clean

runWorker:
	./main/main-worker 127.0.0.1 12366 plugins/worker-plugin.so

runMaster:
	./main/mock-master 12366 ../datasets/test.txt

build:
	(cd plugins && go build -buildmode=plugin worker-plugin.go)
	(cd main && go build main-worker.go)
	(cd main && go build mock-master.go)

clean:
	rm -f main/main-worker
	rm -f main/mock-master
	rm -f plugins/*.so
	rm -rf intermediate