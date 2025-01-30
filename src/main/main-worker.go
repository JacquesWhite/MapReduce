package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"plugin"
	"strconv"

	"github.com/JacquesWhite/MapReduce/worker"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Println("Please provide Master ip and port for connection + WorkerIP + plugin file with Map and Reduce functions")
		_, err := fmt.Fprintf(os.Stderr, "Usage: ./main-worker master_ip master_port worker_ip {file_name}.so\n")
		if err != nil {
			return
		}
		os.Exit(1)
	}

	mapFunc, reduceFunc := loadPlugin(os.Args[4])
	contextWorker := worker.ContextWorker{
		MasterIP:   os.Args[1],
		MasterPort: os.Args[2],
		WorkerIP:   os.Args[3],
		WorkerPort: strconv.Itoa(rand.Intn(60000)),
		MapFunc:    mapFunc,
		ReduceFunc: reduceFunc,
	}

	worker.StartWorkerServer(contextWorker)
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
