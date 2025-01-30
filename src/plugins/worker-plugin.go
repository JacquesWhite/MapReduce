// Package main needed for the plugin to actually build
package main

import (
	"strconv"
	"strings"

	"github.com/JacquesWhite/MapReduce/mr"
)

// Some simple predefined functions for Map and Reduce
// Counting the words inside the input file.
// Contents might be not the whole file, but a part of it,
// which will be predefined by the Master.
// (Master will divide the file and pass it to the Workers)

func Map(f string, contents string) []mr.KeyValue {
	// Split file contents into an array of words.
	words := strings.FieldsFunc(contents, mr.IsSeparator)

	// Create a KeyValue pair for each word.
	var kva []mr.KeyValue
	for _, w := range words {
		kv := mr.KeyValue{Key: w, Value: "1"}
		kva = append(kva, kv)
	}

	return kva
}

func Reduce(key string, values []string) string {
	// Returns the number of occurrences of this word.
	return strconv.Itoa(len(values))
}
