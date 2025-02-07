package worker

import (
	"bufio"
	"context"
	"hash/fnv"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog/log"

	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

type KeyValue struct {
	// Key - what are we mapping
	Key string
	// Value - to what are we mapping to
	Value string
}

type MapFuncT = func(string, string) []KeyValue
type ReduceFuncT = func(string, []string) string

// ByKey Sorting interface for KeyValue intermediate output.
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func IsSeparator(char int32) bool {
	return !unicode.IsLetter(char) && !(char == '-')
}

func hashIdx(key string, nParts int32) int32 {
	// Function to choose the bucket number in which the KeyValue will be emitted.
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		log.Err(err).Msg("Error writing to hash function")
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

func (w *ServiceWorker) changeStatus(status workerpb.CheckStatusResponse_Status) {
	w.statusMx.Lock()
	w.status = status
	w.statusMx.Unlock()
}

func (w *ServiceWorker) readMapInput(intermediateDir, inputFile string) ([]byte, error) {
	log.Info().Msg("Checking for Directory existence, if not creating it")
	err := os.MkdirAll(intermediateDir, os.ModePerm)
	if os.IsExist(err) {
		log.Info().Msgf("The directory named %s exists, nothing created", intermediateDir)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error creating directory: %v", err)
	}

	content, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error reading file: %v", err)
	}
	return content, nil
}

func (w *ServiceWorker) mapResultsToPartitions(mapRes []KeyValue, nPartitions int32) [][]KeyValue {
	kvAll := make([][]KeyValue, nPartitions)

	// Map the results to the partitions (Map mapFunction results into nPartitions buckets)
	for _, kv := range mapRes {
		idx := hashIdx(kv.Key, nPartitions)
		kvAll[idx] = append(kvAll[idx], kv)
	}

	return kvAll
}

func (w *ServiceWorker) saveMapOutput(kvAll [][]KeyValue, intermediateDir string) ([]string, error) {
	partitions := make([]string, len(kvAll))
	for i, kvs := range kvAll {
		filepath := intermediateDir + "/intermediate-" + strconv.Itoa(i)
		partitions[i] = filepath
		f, err := os.Create(filepath)
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
	return partitions, nil
}

func (w *ServiceWorker) readReduceInput(files []string) ([]KeyValue, error) {
	var intermediate []KeyValue
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Reduce: error reading file: %v", err)
		}

		log.Info().Msgf("Reading file: %s", file)
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			kv := KeyValue{Key: parts[0], Value: parts[1]}
			intermediate = append(intermediate, kv)
		}
	}

	log.Info().Msg("Sorting intermediate results")
	sort.Sort(ByKey(intermediate))

	return intermediate, nil
}

func (w *ServiceWorker) processReduce(intermediate []KeyValue, outFile *os.File) error {
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
			return status.Errorf(codes.Internal, "Reduce: error writing to output file: %v", err)
		}

		i = j
	}
	return nil
}

func (w *ServiceWorker) Map(_ context.Context, request *workerpb.MapRequest) (*workerpb.MapResponse, error) {
	log.Info().Msgf("Map request received with file: %s and intermediate directory: %s", request.GetInputFile(), request.GetIntermediateDir())
	w.changeStatus(workerpb.CheckStatusResponse_BUSY)

	content, err := w.readMapInput(request.GetIntermediateDir(), request.GetInputFile())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error reading file: %v", err)
	}

	log.Info().Msg("Invoking Map function on file contents")
	mapRes := w.mapFunc(request.GetInputFile(), string(content))

	kvAll := w.mapResultsToPartitions(mapRes, request.GetNumPartitions())

	// Write the partitioned results to the intermediate files
	partitions, err := w.saveMapOutput(kvAll, request.GetIntermediateDir())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map: error saving map intermediate output: %v", err)
	}

	w.changeStatus(workerpb.CheckStatusResponse_COMPLETED)
	log.Info().Msg("Map request finished")

	return &workerpb.MapResponse{
		TaskId:            request.GetTaskId(),
		IntermediateFiles: partitions,
	}, nil
}

func (w *ServiceWorker) Reduce(_ context.Context, request *workerpb.ReduceRequest) (*workerpb.ReduceResponse, error) {
	log.Info().Msgf("Reduce request received with output file: %s", request.GetOutputFile())
	w.changeStatus(workerpb.CheckStatusResponse_BUSY)

	// Read all files with intermediate output from Maps
	intermediate, err := w.readReduceInput(request.GetIntermediateFiles())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Reduce: error reading intermediate files: %v", err)
	}

	log.Info().Msg("Create the output file (if exists, truncate it)")
	outFile, err := os.Create(request.GetOutputFile())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Reduce: error creating output file: %v", err)
	}

	// Reduce the values with the same key and write them to the output file
	err = w.processReduce(intermediate, outFile)

	w.changeStatus(workerpb.CheckStatusResponse_COMPLETED)
	log.Info().Msg("Reduce request finished")

	return &workerpb.ReduceResponse{
		TaskId: request.GetTaskId(),
	}, nil
}

func (w *ServiceWorker) CheckStatus(_ context.Context, _ *workerpb.CheckStatusRequest) (*workerpb.CheckStatusResponse, error) {
	log.Info().Msg("CheckStatus request received")

	w.statusMx.Lock()
	workerStatus := w.status
	log.Info().Msgf("Worker status: %v", workerStatus)
	// Masters collects the status, so we can reset it to IDLE
	if workerStatus == workerpb.CheckStatusResponse_COMPLETED {
		w.status = workerpb.CheckStatusResponse_IDLE
	}
	w.statusMx.Unlock()
	log.Info().Msg("CheckStatus request finished")

	return &workerpb.CheckStatusResponse{Status: workerStatus}, nil
}
