package main

import (
	"flag"
	"log"
	"plugin"

	"github.com/JacquesWhite/MapReduce/runner/worker_startup"
	"github.com/JacquesWhite/MapReduce/worker"
)

func main() {
	workerPort := flag.String("w_port", "50001", "The worker port")
	workerIP := flag.String("w_ip", "localhost", "The worker IP")
	masterPort := flag.String("m_port", "50051", "The master port")
	masterIP := flag.String("m_ip", "localhost", "The master IP")
	pluginFile := flag.String("plugin", "", "The plugin file with Map and Reduce functions")
	flag.Parse()

	mapFunc, reduceFunc := loadPlugin(*pluginFile)
	contextWorker := worker_startup.ContextWorker{
		MasterIP:   *masterIP,
		MasterPort: *masterPort,
		WorkerIP:   *workerIP,
		WorkerPort: *workerPort,
		MapFunc:    mapFunc,
		ReduceFunc: reduceFunc,
	}

	worker_startup.StartWorkerServer(contextWorker)
}

// Load the Map and Reduce functions for Worker for further use
// Even if we have Map and Reduce functions predefined, this can be
// useful for further expansion of the project.
func loadPlugin(filename string) (worker.MapFuncT, worker.ReduceFuncT) {
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}

	lookupMapFunc, err := p.Lookup("Map")
	if err != nil {
		log.Fatalf("cannot find Map function in %v", filename)
	}
	mapFunc := lookupMapFunc.(worker.MapFuncT)

	lookupReduceFunc, err := p.Lookup("Reduce")
	if err != nil {
		log.Fatalf("cannot find Reduce in %v", filename)
	}
	reduceFunc := lookupReduceFunc.(worker.ReduceFuncT)

	return mapFunc, reduceFunc
}
