package worker

import (
	"bufio"
	"context"
	"hash/fnv"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

type KeyValue struct {
	// Key - what are we mapping
	Key string
	// Value - to what are we mapping to
	Value string
}

// ByKey Sorting interface for KeyValue intermediate output.
type ByKey []KeyValue

func (a ByKey) Len() int               { return len(a) }
func (a ByKey) Swap(i int, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i int, j int) bool { return a[i].Key < a[j].Key }

func IsSeparator(char int32) bool {
	return !unicode.IsLetter(char)
}

func ihashIdx(key string, nParts int32) int32 {
	// Function to choose the bucket number in which the KeyValue will be emitted.
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0
	}
	return int32(h.Sum32()&0x7fffffff) % nParts
}

type ServiceWorker struct {
	workerpb.UnimplementedWorkerServer

	mapFunc    MapFuncT
	reduceFunc ReduceFuncT
	statusMx   sync.Mutex
	status     workerpb.CheckStatusResponse_Status
}

func NewWorkerService(m MapFuncT, r ReduceFuncT) *ServiceWorker {
	return &ServiceWorker{
		mapFunc:    m,
		reduceFunc: r,
		status:     workerpb.CheckStatusResponse_IDLE,
	}
}

func (w *ServiceWorker) Map(_ context.Context, request *workerpb.MapRequest) (*workerpb.MapResponse, error) {
	log.Println("Map request received with file:", request.GetInputFile(), "and intermediate directory:", request.GetIntermediateDir())
	w.statusMx.Lock()
	w.status = workerpb.CheckStatusResponse_BUSY
	w.statusMx.Unlock()

	log.Println("Checking for Directory existence, if not creating it")
	err := os.MkdirAll(request.GetIntermediateDir(), os.ModePerm)
	if os.IsExist(err) {
		log.Println("The directory named ", request.GetIntermediateDir(), " exists, nothing created")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error creating directory: %v", err)
	}

	content, err := os.ReadFile(request.GetInputFile())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error reading file: %v", err)
	}

	log.Println("Invoking Map function on file contents")
	mapRes := w.mapFunc(request.GetInputFile(), string(content))
	kvAll := make([][]KeyValue, request.GetNumPartitions())

	// Map the results to the partitions (Map mapFunction results into nPartitions buckets)
	for _, kv := range mapRes {
		idx := ihashIdx(kv.Key, request.GetNumPartitions())
		kvAll[idx] = append(kvAll[idx], kv)
	}

	// Write the partitioned results to the intermediate files
	for i, kvs := range kvAll {
		f, err := os.Create(request.GetIntermediateDir() + "/intermediate-" + strconv.Itoa(i))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Map: error creating intermediate file: %v", err)
		}

		for _, kv := range kvs {
			_, err := f.WriteString(kv.Key + " " + kv.Value + "\n")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Map: error writing to intermediate file: %v", err)
			}
		}

		err = f.Close()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Map: error closing intermediate file: %v", err)
		}
	}

	w.statusMx.Lock()
	w.status = workerpb.CheckStatusResponse_IDLE
	w.statusMx.Unlock()
	log.Println("Map finished")

	return &workerpb.MapResponse{}, nil
}

func (w *ServiceWorker) Reduce(_ context.Context, request *workerpb.ReduceRequest) (*workerpb.ReduceResponse, error) {
	log.Println("Reduce request received with output file:", request.GetOutputFile())
	w.statusMx.Lock()
	w.status = workerpb.CheckStatusResponse_BUSY
	w.statusMx.Unlock()

	// Read all files with intermediate output from Maps
	var intermediate []KeyValue
	for _, file := range request.GetIntermediateFiles() {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Reduce: error reading file: %v", err)
		}

		log.Println("Reading file:", file)
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			kv := KeyValue{Key: parts[0], Value: parts[1]}
			intermediate = append(intermediate, kv)
		}
	}

	log.Println("Sorting intermediate results")
	sort.Sort(ByKey(intermediate))

	log.Println("Create the output file (if exists, truncate it)")
	outFile, err := os.Create(request.GetOutputFile())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Reduce: error creating output file: %v", err)
	}

	// Reduce the values with the same key and write them to the output file
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}

		var values []string
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}

		output := w.reduceFunc(intermediate[i].Key, values)
		_, err := outFile.WriteString(intermediate[i].Key + " " + output + "\n")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Reduce: error writing to output file: %v", err)
		}

		i = j
	}

	w.statusMx.Lock()
	w.status = workerpb.CheckStatusResponse_IDLE
	w.statusMx.Unlock()
	log.Println("Reduce request finished")

	return &workerpb.ReduceResponse{}, nil
}

func (w *ServiceWorker) CheckStatus(_ context.Context, _ *workerpb.CheckStatusRequest) (*workerpb.CheckStatusResponse, error) {
	log.Println("CheckStatus request received")

	w.statusMx.Lock()
	workerStatus := w.status
	w.statusMx.Unlock()
	log.Println("CheckStatus request finished")

	return &workerpb.CheckStatusResponse{Status: workerStatus}, nil
}
