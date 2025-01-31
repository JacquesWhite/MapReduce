package main

// MapReduce functions that always crash

import (
	"os"
	//"time"

	"github.com/JacquesWhite/MapReduce/worker"
)

func Map(_ string, _ string) []worker.KeyValue {
	os.Exit(1)
	//time.Sleep(10 * time.Second)
	return []worker.KeyValue{}
}

func Reduce(_ string, _ []string) string {
	//time.Sleep(10 * time.Second)
	os.Exit(1)
	return ""
}
