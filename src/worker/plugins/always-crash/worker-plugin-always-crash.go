package main

// MapReduce plugin that always crashes.

import (
	"os"

	"github.com/JacquesWhite/MapReduce/worker/worker_utils"
)

func Map(_ string, _ string) []worker_utils.KeyValue {
	os.Exit(1)
	return []worker_utils.KeyValue{}
}

func Reduce(_ string, _ []string) string {
	os.Exit(1)
	return ""
}
