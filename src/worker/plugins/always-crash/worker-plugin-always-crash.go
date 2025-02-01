package main

// MapReduce plugin that always crashes.

import (
	"os"

	"github.com/JacquesWhite/MapReduce/worker"
)

func Map(_ string, _ string) []worker.KeyValue {
	os.Exit(1)
	return []worker.KeyValue{}
}

func Reduce(_ string, _ []string) string {
	os.Exit(1)
	return ""
}
