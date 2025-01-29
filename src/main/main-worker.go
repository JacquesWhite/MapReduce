package main

import (
	"fmt"
	"github.com/JacquesWhite/MapReduce/src/mr"
	"log"
	"os"
	"plugin"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Please provide Master ip and port for connection + plugin file with Map and Reduce functions")
		_, err := fmt.Fprintf(os.Stderr, "Usage: ./main-worker master_ip master_port {file_name}.so\n")
		if err != nil {
			return
		}
		os.Exit(1)
	}

	mapFunc, reduceFunc := loadPlugin(os.Args[3])
	context := mr.WorkerContext{
		MasterIP:   os.Args[1],
		MasterPort: os.Args[2],
		MapFunc:    mapFunc,
		ReduceFunc: reduceFunc,
	}

	mr.WorkerMain(context)
}

// Load the Map and Reduce functions for Worker for further use
// Even if we have Map and Reduce functions predefined, this can be
// useful for further expansion of the project.
func loadPlugin(filename string) (func(string, string) []mr.KeyValue, func(string, []string) string) {
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}

	lookupMapFunc, err := p.Lookup("Map")
	if err != nil {
		log.Fatalf("cannot find Map function in %v", filename)
	}
	mapFunc := lookupMapFunc.(func(string, string) []mr.KeyValue)

	lookupReduceFunc, err := p.Lookup("Reduce")
	if err != nil {
		log.Fatalf("cannot find Reduce in %v", filename)
	}
	reduceFunc := lookupReduceFunc.(func(string, []string) string)

	return mapFunc, reduceFunc
}
