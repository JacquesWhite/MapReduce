package mr

import (
	"hash/fnv"
	"unicode"
)

// KeyValue struct - Map functions return an array of KeyValue.
type KeyValue struct {
	// Key - what are we mapping
	Key string
	// Value - to what are we mapping to
	Value string
}

// ByKey Sorting interface for KeyValue for intermediate output.
type ByKey []KeyValue

func (a ByKey) Len() int               { return len(a) }
func (a ByKey) Swap(i int, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i int, j int) bool { return a[i].Key < a[j].Key }

// IsSeparator - Check if the char is a separator.
func IsSeparator(char int32) bool {
	return !unicode.IsLetter(char)
}

// Use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0
	}
	return int(h.Sum32() & 0x7fffffff)
}
