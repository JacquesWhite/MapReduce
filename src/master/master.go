package master

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

const (
	intermediateDirName = "/intermediate"
	outputDirName       = "/output"
)

type TaskState int

// Task states
const (
	NotAssigned TaskState = iota
	Assigned
	Completed
	Invalid
)

type WorkerState int

const (
	Idle WorkerState = iota
	Processing
	Done
	Failure
)

func (ts TaskState) String() string {
	switch ts {
	case NotAssigned:
		return "NotAssigned"
	case Assigned:
		return "Assigned"
	case Completed:
		return "Completed"
	case Invalid:
		return "Invalid"
	default:
		return "Unknown"
	}
}

type MapTask struct {
	id              int
	worker          *workerpb.WorkerClient
	inputFile       string
	intermediateDir string
	state           TaskState
	mx              sync.Mutex
}

type ReduceTask struct {
	id         int
	worker     *workerpb.WorkerClient
	inputFiles []string
	outputFile string
	state      TaskState
	mx         sync.Mutex
}

type RunnableTask interface {
	RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service)
	CheckState() TaskState
	ChangeState(state TaskState)
}

type Service struct {
	masterpb.UnimplementedMasterServer

	numberOfPartitions    int32
	workers               map[*workerpb.WorkerClient]bool
	registeredAddress     map[string]bool
	inputDir              string
	intermediateDir       string
	outputDir             string
	mapTaskToProcess      int
	unprocessedMapResults int
	mapTasks              []*MapTask
	mapResults            chan *workerpb.MapResponse
	reduceTaskToProcess   int
	reduceTasks           []*ReduceTask
	reduceResults         chan *workerpb.ReduceResponse
	mx                    sync.Mutex
}

func createWorkerClient(workerAddress *masterpb.WorkerAddress) (workerpb.WorkerClient, error) {
	address := workerAddress.GetIp() + ":" + workerAddress.GetPort()
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create connection to worker: %v", err)
	}
	return workerpb.NewWorkerClient(conn), nil
}

func (s *Service) createMapTasks() error {
	files, err := os.ReadDir(s.inputDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Creating map tasks")

	for i, file := range files {
		taskDir := fmt.Sprintf("%s/map_%d", s.intermediateDir, i)
		err = os.MkdirAll(taskDir, os.ModePerm)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
		}
		s.mapTasks = append(s.mapTasks, &MapTask{
			id:              i,
			worker:          nil,
			inputFile:       s.inputDir + "/" + file.Name(),
			intermediateDir: taskDir,
			state:           NotAssigned,
		})
		log.Printf("Created map task: %v", s.mapTasks[i])
	}
	s.mapTaskToProcess = len(s.mapTasks)
	s.unprocessedMapResults = len(s.mapTasks)
	s.mapResults = make(chan *workerpb.MapResponse, len(s.mapTasks))
	log.Printf("Created map tasks")
	return nil
}

func (s *Service) prepareReduceInputFiles() [][]string {
	inputFiles := make([][]string, s.numberOfPartitions)
	for s.unprocessedMapResults > 0 {
		res := <-s.mapResults
		log.Printf("Processing map result: %v", res)
		log.Printf("Task id: %d, ended with status: %s", res.GetTaskId(), s.mapTasks[res.GetTaskId()].CheckState().String())
		if s.mapTasks[res.GetTaskId()].CheckState() != Completed {
			log.Printf("Map task %d has not been completed", res.GetTaskId())
			continue
		}
		for partition, file := range res.GetIntermediateFiles() {
			log.Printf("Partition: %d, file: %s", partition, file)
			inputFiles[partition] = append(inputFiles[partition], file)
		}
		s.unprocessedMapResults--
	}
	return inputFiles
}

func (s *Service) prepareResponse() (*masterpb.MapReduceResponse, error) {
	log.Printf("Preparing response")
	outputFiles := make([]string, s.numberOfPartitions)
	filledOutputFiles := 0
	for filledOutputFiles != int(s.numberOfPartitions) {
		res := <-s.reduceResults
		log.Printf("Processing reduce result: %v", res)
		log.Printf("Task id: %d, ended with status: %s", res.GetTaskId(), s.reduceTasks[res.GetTaskId()].CheckState().String())
		if s.reduceTasks[res.GetTaskId()].CheckState() != Completed {
			continue
		}
		outputFiles[res.GetTaskId()] = s.reduceTasks[res.GetTaskId()].outputFile
		filledOutputFiles++
	}
	log.Printf("Prepared response")
	return &masterpb.MapReduceResponse{
		OutputFiles: outputFiles,
	}, nil
}

func (s *Service) createReduceTasks() error {
	log.Printf("Creating reduce tasks")
	if s.mapTaskToProcess != 0 {
		return status.Errorf(codes.Internal, "All map tasks should have been processed before creating reduce tasks")
	}
	err := os.MkdirAll(s.outputDir, os.ModePerm)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
	}
	log.Printf("Preparing reduce input files")
	inputFiles := s.prepareReduceInputFiles()
	log.Printf("Prepared reduce input files")
	for i := 0; i < int(s.numberOfPartitions); i++ {
		s.reduceTasks = append(s.reduceTasks, &ReduceTask{
			id:         i,
			worker:     nil,
			inputFiles: inputFiles[i],
			outputFile: fmt.Sprintf("%s/%d", s.outputDir, i),
			state:      NotAssigned,
		})
		log.Printf("Created reduce task: %v", s.reduceTasks[i])
	}
	log.Printf("Created reduce tasks")
	s.reduceTaskToProcess = len(s.reduceTasks)
	return nil
}

func (t *MapTask) CheckState() TaskState {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.state
}

func (t *ReduceTask) CheckState() TaskState {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.state
}

func (t *MapTask) ChangeState(state TaskState) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.state = state
}

func (t *ReduceTask) ChangeState(state TaskState) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.state = state
}

func (t *MapTask) CheckWorker() *workerpb.WorkerClient {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.worker
}

func (t *ReduceTask) CheckWorker() *workerpb.WorkerClient {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.worker
}

func (t *ReduceTask) RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service) {
	log.Printf("Assigning reduce task %d to worker %v", t.id, worker)
	service.workers[worker] = false
	t.worker = worker
	t.state = Assigned
	inputFiles := t.inputFiles
	outputFile := t.outputFile
	service.mx.Unlock()
	go func() {
		res, err := (*worker).Reduce(ctx, &workerpb.ReduceRequest{
			IntermediateFiles: inputFiles,
			OutputFile:        outputFile,
			TaskId:            int32(t.id),
		})
		if err != nil {
			log.Printf("Error while processing reduce task: %v", err)
		}
		service.reduceResults <- res
		log.Printf("Worker completed reduce task %d", t.id)
	}()
}

func (t *MapTask) RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service) {
	log.Printf("Assigning map task %d to worker %v", t.id, worker)
	service.workers[worker] = false
	t.worker = worker
	t.ChangeState(Assigned)
	inputFile := t.inputFile
	intermediateDir := t.intermediateDir
	service.mx.Unlock()
	go func() {
		res, err := (*worker).Map(ctx, &workerpb.MapRequest{
			InputFile:       inputFile,
			IntermediateDir: intermediateDir,
			NumPartitions:   service.numberOfPartitions,
			TaskId:          int32(t.id),
		})
		if err != nil {
			log.Printf("Error while processing map task: %v", err)
		}
		service.mapResults <- res
		log.Printf("Worker completed map task %d", t.id)
	}()
}

func (t *MapTask) CheckWorkerStatus(ctx context.Context) WorkerState {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	res, err := (*t.worker).CheckStatus(ctxWithTimeout, &workerpb.CheckStatusRequest{})
	if err != nil {
		log.Printf("Error while checking worker status, for task %d: %v", t.id, err)
		return Failure
	}
	if res.GetStatus() == workerpb.CheckStatusResponse_IDLE {
		return Idle
	} else if res.GetStatus() == workerpb.CheckStatusResponse_BUSY {
		return Processing
	} else if res.GetStatus() == workerpb.CheckStatusResponse_COMPLETED {
		return Done
	}
	return Failure
}

func (t *ReduceTask) CheckWorkerStatus(ctx context.Context) WorkerState {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	res, err := (*t.worker).CheckStatus(ctxWithTimeout, &workerpb.CheckStatusRequest{})
	if err != nil {
		log.Printf("Error while checking worker status, for task %d: %v", t.id, err)
		return Failure
	}
	if res.GetStatus() == workerpb.CheckStatusResponse_IDLE {
		return Idle
	} else if res.GetStatus() == workerpb.CheckStatusResponse_BUSY {
		return Processing
	} else if res.GetStatus() == workerpb.CheckStatusResponse_COMPLETED {
		return Done
	}
	return Failure
}

func (s *Service) createMapTaskCopy(idx int) error {
	s.mx.Lock()
	task := s.mapTasks[idx]
	newId := len(s.mapTasks)
	taskDir := fmt.Sprintf("%s/map_%d", s.intermediateDir, newId)
	s.mx.Unlock()
	err := os.MkdirAll(taskDir, os.ModePerm)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
	}
	newTask := &MapTask{
		id:              newId,
		worker:          nil,
		inputFile:       task.inputFile,
		intermediateDir: taskDir,
		state:           NotAssigned,
	}
	s.mx.Lock()
	s.mapTasks = append(s.mapTasks, newTask)
	s.mx.Unlock()
	return nil
}

func (s *Service) createReduceTaskCopy(idx int) error {
	s.mx.Lock()
	task := s.reduceTasks[idx]
	newId := len(s.reduceTasks)
	s.mx.Unlock()
	newTask := &ReduceTask{
		id:         newId,
		worker:     nil,
		inputFiles: task.inputFiles,
		outputFile: fmt.Sprintf("%s/%d", s.outputDir, newId),
		state:      NotAssigned,
	}
	s.mx.Lock()
	s.reduceTasks = append(s.reduceTasks, newTask)
	s.mx.Unlock()
	return nil
}

func (s *Service) monitorMapTaskExecution(ctx context.Context, idx int) error {
	s.mx.Lock()
	task := s.mapTasks[idx]
	s.mx.Unlock()
	workerStatus := task.CheckWorkerStatus(ctx)
	//log.Printf("Worker status: %v", workerStatus)
	switch workerStatus {
	case Failure:
		{
			// Worker is not processing the task - reassign the task
			task.ChangeState(Invalid)
			err := s.createMapTaskCopy(idx)
			if err != nil {
				log.Printf("Monitor map task execution error while creating map task copy: %v", err)
				return err
			}
		}
	case Idle, Processing:
		{
			//	Worker is processing the task - come back later
		}
	case Done:
		{
			// Worker has completed the task
			log.Printf("1Worker has completed map task %d", task.id)
			task.ChangeState(Completed)
			s.mx.Lock()
			log.Printf("2Worker has completed map task %d", task.id)
			s.mapTaskToProcess--
			s.workers[task.CheckWorker()] = true
			s.mx.Unlock()
		}
	default:
		panic("unhandled default case")
	}
	return nil
}

func (s *Service) monitorReduceTaskExecution(ctx context.Context, idx int) error {
	s.mx.Lock()
	task := s.reduceTasks[idx]
	s.mx.Unlock()
	workerStatus := task.CheckWorkerStatus(ctx)

	switch workerStatus {
	case Failure:
		{
			// Worker is not processing the task - reassign the task
			task.ChangeState(Invalid)
			err := s.createReduceTaskCopy(idx)
			if err != nil {
				log.Printf("Monitor map task execution error while creating map task copy: %v", err)
				return err
			}
		}
	case Idle, Processing:
		{
			//	Worker is processing the task - come back later
		}
	case Done:
		{
			// Worker has completed the task
			log.Printf("Master noticed that worker has completed reduce task %d", task.id)
			task.ChangeState(Completed)
			s.mx.Lock()
			s.reduceTaskToProcess--
			s.workers[task.CheckWorker()] = true
			s.mx.Unlock()
		}
	default:
		panic("unhandled default case")
	}
	return nil
}

func (s *Service) assignTaskToWorker(ctx context.Context, task RunnableTask) {
	for worker := range s.workers {
		s.mx.Lock()
		available := s.workers[worker]
		if available {
			task.RunTask(ctx, worker, s)
			break
		} else {
			s.mx.Unlock()
		}
	}
}

func (s *Service) processReduceTasks(ctx context.Context) error {
	log.Printf("Processing reduce tasks")
	for s.reduceTaskToProcess > 0 {
		s.mx.Lock()
		numberOfTasks := len(s.reduceTasks)
		s.mx.Unlock()
		for idx := range numberOfTasks {
			s.mx.Lock()
			task := s.reduceTasks[idx]
			s.mx.Unlock()
			if task.CheckState() == NotAssigned {
				s.assignTaskToWorker(ctx, task)
			} else if task.CheckState() == Assigned {
				err := s.monitorReduceTaskExecution(ctx, idx)
				if err != nil {
					return status.Errorf(codes.Internal, "Process reduce error while monitoring reduce task execution: %v", err)
				}
			}
		}
	}
	log.Printf("Completed processing reduce tasks")
	return nil
}

func (s *Service) processMapTasks(ctx context.Context) error {
	log.Printf("Processing map tasks")
	for s.mapTaskToProcess > 0 {
		s.mx.Lock()
		numberOfTasks := len(s.mapTasks)
		s.mx.Unlock()
		for idx := range numberOfTasks {
			s.mx.Lock()
			task := s.mapTasks[idx]
			s.mx.Unlock()
			if task.CheckState() == NotAssigned {
				s.assignTaskToWorker(ctx, task)
			} else if task.CheckState() == Assigned {
				err := s.monitorMapTaskExecution(ctx, idx)
				if err != nil {
					return status.Errorf(codes.Internal, "Process map error while monitoring map task execution: %v", err)
				}
			}
		}
	}
	log.Printf("Completed processing map tasks")
	return nil
}

func NewService() *Service {
	return &Service{
		workers: make(map[*workerpb.WorkerClient]bool),
	}
}

func (s *Service) RegisterWorker(ctx context.Context, req *masterpb.RegisterWorkerRequest) (*masterpb.RegisterWorkerResponse, error) {
	log.Printf("Received RegisterWorkers request: %v", req)
	address := req.GetWorkerAddress().GetIp() + ":" + req.GetWorkerAddress().GetPort()
	s.mx.Lock()
	addressAlreadyExists := s.registeredAddress[address]
	s.mx.Unlock()
	if addressAlreadyExists {
		log.Printf("Worker already registered, no need to create new connection: %v", req.GetWorkerAddress())
		return &masterpb.RegisterWorkerResponse{}, nil
	}
	workerClient, err := createWorkerClient(req.GetWorkerAddress())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: failed to create worker client: %v", err)
	}
	res, err := workerClient.CheckStatus(ctx, &workerpb.CheckStatusRequest{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: failed to check worker status: %v", err)
	}
	if res.GetStatus() != workerpb.CheckStatusResponse_IDLE {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: worker is not ready to accept connections")
	}

	s.mx.Lock()
	s.workers[&workerClient] = true
	s.mx.Unlock()
	log.Printf("Registered successfully worker: %v", req.GetWorkerAddress())
	return &masterpb.RegisterWorkerResponse{}, nil
}

func (s *Service) cleanup() error {
	log.Printf("Cleaning up")
	err := os.RemoveAll(s.intermediateDir)
	if err != nil {
		return err
	}
	s.mapTasks = nil
	s.mapResults = nil
	s.mapTaskToProcess = 0
	s.reduceTasks = nil
	s.reduceResults = nil
	s.reduceTaskToProcess = 0
	return nil
}

func (s *Service) InitializeMapReduce(req *masterpb.MapReduceRequest) error {
	s.numberOfPartitions = req.GetNumPartitions()
	if s.numberOfPartitions <= 0 {
		return status.Errorf(codes.InvalidArgument, "Number of partitions should be greater than 0")
	}
	s.inputDir = req.GetInputDir()
	s.intermediateDir = req.GetWorkingDir() + intermediateDirName
	s.outputDir = req.GetWorkingDir() + outputDirName
	s.reduceResults = make(chan *workerpb.ReduceResponse, s.numberOfPartitions)
	return nil
}

func (s *Service) MapReduce(ctx context.Context, req *masterpb.MapReduceRequest) (*masterpb.MapReduceResponse, error) {
	log.Printf("Received MapReduce request: %v", req)
	err := s.InitializeMapReduce(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to initialize map reduce: %v", err)
	}

	err = s.createMapTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create map tasks: %v", err)
	}
	err = s.processMapTasks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to process map tasks: %v", err)
	}

	err = s.createReduceTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create reduce tasks: %v", err)
	}
	err = s.processReduceTasks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to process reduce tasks: %v", err)
	}

	res, err := s.prepareResponse()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to prepare response: %v", err)
	}

	err = s.cleanup()
	if err != nil {
		return nil, err
	}

	log.Printf("Completed map reduce job")
	return res, nil
}
