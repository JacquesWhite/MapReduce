package main

import (
	"flag"
	"log"

	"github.com/rs/zerolog"

	"github.com/JacquesWhite/MapReduce/worker/worker_utils"
	"github.com/JacquesWhite/MapReduce/worker/worker_utils/startup_utils"
)

func main() {
	workerPort := flag.String("w_port", "50001", "The worker port")
	workerIP := flag.String("w_ip", "localhost", "The worker IP")
	masterPort := flag.String("m_port", "50051", "The master port")
	masterIP := flag.String("m_ip", "localhost", "The master IP")
	pluginFile := flag.String("plugin", "", "The plugin file with Map and Reduce functions")

	// Set logging settings
	zerolog.LevelFieldName = "severity"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Parse the flags
	flag.Parse()

	mapFunc, reduceFunc, err := worker_utils.LoadPlugin(*pluginFile)
	if err != nil {
		log.Fatalf("Worker startup: %v", err)
	}
	contextWorker := startup_utils.ContextWorker{
		MasterIP:   *masterIP,
		MasterPort: *masterPort,
		WorkerIP:   *workerIP,
		WorkerPort: *workerPort,
		MapFunc:    mapFunc,
		ReduceFunc: reduceFunc,
	}

	startup_utils.StartWorkerServer(contextWorker)
}
