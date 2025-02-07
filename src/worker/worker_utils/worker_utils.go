package worker_utils

import (
	"hash/fnv"
	"plugin"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog/log"
)

type KeyValue struct {
	// Key - what are we mapping
	Key string
	// Value - to what are we mapping to
	Value string
}

// ByKey Sorting interface for KeyValue intermediate output.
type ByKey []KeyValue

type MapFuncT = func(string, string) []KeyValue
type ReduceFuncT = func(string, []string) string

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func IsSeparator(char int32) bool {
	return !unicode.IsLetter(char) && !(char == '-')
}

func HashIdx(key string, nParts int32) int32 {
	// Function to choose the bucket number in which the KeyValue will be emitted.
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		log.Err(err).Msg("Error writing to hash function")
		return 0
	}
	return int32(h.Sum32()&0x7fffffff) % nParts
}

// LoadPlugin loads the Map and Reduce functions for Worker for further use
// Even if we have Map and Reduce functions predefined, this can be
// useful for further expansion of the project.
func LoadPlugin(filename string) (MapFuncT, ReduceFuncT, error) {
	p, err := plugin.Open(filename)
	if err != nil {
		log.Err(err).Msgf("cannot load plugin %v", filename)
		return nil, nil, status.Errorf(codes.Internal, "Load plugin: cannot load plugin %v", filename)
	}

	lookupMapFunc, err := p.Lookup("Map")
	if err != nil {
		log.Err(err).Msgf("cannot find Map function in %v", filename)
		return nil, nil, status.Errorf(codes.Internal, "Load plugin: cannot find Map function in %v", filename)
	}
	mapFunc := lookupMapFunc.(MapFuncT)

	lookupReduceFunc, err := p.Lookup("Reduce")
	if err != nil {
		log.Err(err).Msgf("cannot find Reduce function in %v", filename)
		return nil, nil, status.Errorf(codes.Internal, "Load plugin: cannot find Reduce function in %v", filename)
	}
	reduceFunc := lookupReduceFunc.(ReduceFuncT)

	return mapFunc, reduceFunc, nil
}
