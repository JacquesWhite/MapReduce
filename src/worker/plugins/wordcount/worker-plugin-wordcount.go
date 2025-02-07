// Package main needed for the plugin to actually build
package main

import (
	"strconv"
	"strings"

	"github.com/JacquesWhite/MapReduce/worker/worker_utils"
)

// Some simple predefined functions for Map and Reduce
// Counting the words inside the input file.
// Contents might be not the whole file, but a part of it,
// which will be predefined by the Master.
// (Master will divide the file and pass it to the Workers)

func Map(_ string, contents string) []worker_utils.KeyValue {
	// Split file contents into an array of words.
	words := strings.FieldsFunc(contents, worker_utils.IsSeparator)

	// Create a KeyValue pair for each word.
	var kva []worker_utils.KeyValue
	for _, w := range words {
		kv := worker_utils.KeyValue{Key: strings.ToLower(w), Value: "1"}
		kva = append(kva, kv)
	}

	return kva
}

func Reduce(_ string, values []string) string {
	// Returns the number of occurrences of this word.
	return strconv.Itoa(len(values))
}
