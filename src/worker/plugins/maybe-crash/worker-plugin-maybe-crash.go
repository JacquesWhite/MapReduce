package main

// MapReduce plugin that works the same, as the wordcount plugin,
// but sometimes it crashes and sometimes takes a long time to exec,
// to test MapReduce ability to recover / parse execution.

import (
	crand "crypto/rand"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JacquesWhite/MapReduce/worker"
)

func maybeCrash() {
	randMax := big.NewInt(10)
	rr, err := crand.Int(crand.Reader, randMax)
	if err != nil {
		// If crand.Int failed -> crash
		os.Exit(1)
	}

	gen := rr.Int64()
	if gen < 2 {
		// For generated 0 and 1 -> crash
		os.Exit(1)
	} else {
		// For generated [2..10] -> sleep
		time.Sleep(time.Duration(gen) * time.Second)
	}
}

func Map(_ string, contents string) []worker.KeyValue {
	// Call crash or sleep
	maybeCrash()

	// Split file contents into an array of words.
	words := strings.FieldsFunc(contents, worker.IsSeparator)

	// Create a KeyValue pair for each word.
	var kva []worker.KeyValue
	for _, w := range words {
		kv := worker.KeyValue{Key: strings.ToLower(w), Value: "1"}
		kva = append(kva, kv)
	}

	return kva
}

func Reduce(_ string, values []string) string {
	// Call crash or sleep
	maybeCrash()

	// Returns the number of occurrences of this word.
	return strconv.Itoa(len(values))
}
