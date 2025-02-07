package master

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog/log"

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

type mapTask struct {
	id              int
	worker          *workerpb.WorkerClient
	inputFile       string
	intermediateDir string
	pluginFilepath  string
	state           TaskState
	mx              sync.Mutex
}

type reduceTask struct {
	id             int
	worker         *workerpb.WorkerClient
	inputFiles     []string
	outputFile     string
	pluginFilepath string
	state          TaskState
	mx             sync.Mutex
}

type RunnableTask interface {
	RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service)
	CheckState() TaskState
	ChangeState(state TaskState)
	CheckWorker() *workerpb.WorkerClient
	CheckId() int
}

type Service struct {
	masterpb.UnimplementedMasterServer

	numberOfPartitions    int32
	workers               map[*workerpb.WorkerClient]bool
	registeredAddress     map[string]bool
	inputDir              string
	intermediateDir       string
	outputDir             string
	pluginFilePath        string
	mapTaskToProcess      int
	unprocessedMapResults int
	mapTasks              []*mapTask
	mapResults            chan *workerpb.MapResponse
	reduceTaskToProcess   int
	reduceTasks           []*reduceTask
	reduceResults         chan *workerpb.ReduceResponse
	mx                    sync.Mutex
	// Only one MapReduce job can be processed at a time
	mr sync.Mutex
}

func (t *mapTask) CheckState() TaskState {
	return t.state
}

func (t *reduceTask) CheckState() TaskState {
	return t.state
}

func (t *mapTask) ChangeState(state TaskState) {
	t.state = state
}

func (t *reduceTask) ChangeState(state TaskState) {
	t.state = state
}

func (t *mapTask) CheckWorker() *workerpb.WorkerClient {
	return t.worker
}

func (t *reduceTask) CheckWorker() *workerpb.WorkerClient {
	return t.worker
}

func (t *mapTask) CheckId() int {
	return t.id
}

func (t *reduceTask) CheckId() int {
	return t.id
}

func logLineSeparator() {
	log.Printf("------------------------------------------------------------------------------")
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
		log.Fatal().Err(err).Msg("Failed to read input directory")
	}
	log.Info().Msg("Creating map tasks")

	for i, file := range files {
		taskDir := fmt.Sprintf("%s/map_%d", s.intermediateDir, i)
		err = os.MkdirAll(taskDir, os.ModePerm)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
		}
		s.mapTasks = append(s.mapTasks, &mapTask{
			id:              i,
			worker:          nil,
			inputFile:       s.inputDir + "/" + file.Name(),
			intermediateDir: taskDir,
			state:           NotAssigned,
			pluginFilepath:  s.pluginFilePath,
		})
		log.Info().Msgf("Created map task: %v", s.mapTasks[i])
	}
	s.mapTaskToProcess = len(s.mapTasks)
	s.unprocessedMapResults = len(s.mapTasks)
	s.mapResults = make(chan *workerpb.MapResponse, len(s.mapTasks))
	log.Info().Msg("Created map tasks")
	return nil
}

func (s *Service) prepareReduceInputFiles() [][]string {
	inputFiles := make([][]string, s.numberOfPartitions)
	for s.unprocessedMapResults > 0 {
		res := <-s.mapResults
		log.Info().Msgf("Processing map result: %v", res)
		log.Info().Msgf("Task id: %d, ended with status: %s", res.GetTaskId(), s.mapTasks[res.GetTaskId()].CheckState().String())
		if s.mapTasks[res.GetTaskId()].CheckState() != Completed {
			log.Info().Msgf("Map task %d has not been completed", res.GetTaskId())
			continue
		}
		for partition, file := range res.GetIntermediateFiles() {
			inputFiles[partition] = append(inputFiles[partition], file)
		}
		s.unprocessedMapResults--
	}
	return inputFiles
}

func (s *Service) prepareResponse() (*masterpb.MapReduceResponse, error) {
	log.Info().Msg("Preparing response")
	outputFiles := make([]string, s.numberOfPartitions)
	filledOutputFiles := 0
	for filledOutputFiles != int(s.numberOfPartitions) {
		res := <-s.reduceResults
		log.Info().Msgf("Processing reduce result: %v", res)
		log.Info().Msgf("Task id: %d, ended with status: %s", res.GetTaskId(), s.reduceTasks[res.GetTaskId()].CheckState().String())
		if s.reduceTasks[res.GetTaskId()].CheckState() != Completed {
			continue
		}
		outputFiles[res.GetTaskId()] = s.reduceTasks[res.GetTaskId()].outputFile
		filledOutputFiles++
	}
	log.Info().Msg("Prepared response")
	return &masterpb.MapReduceResponse{
		OutputFiles: outputFiles,
	}, nil
}

func (s *Service) createReduceTasks() error {
	log.Info().Msg("Creating reduce tasks")
	if s.mapTaskToProcess != 0 {
		return status.Errorf(codes.Internal, "All map tasks should have been processed before creating reduce tasks")
	}
	err := os.MkdirAll(s.outputDir, os.ModePerm)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
	}
	log.Info().Msg("Preparing reduce input files")
	inputFiles := s.prepareReduceInputFiles()
	log.Info().Msg("Prepared reduce input files")
	for i := 0; i < int(s.numberOfPartitions); i++ {
		s.reduceTasks = append(s.reduceTasks, &reduceTask{
			id:             i,
			worker:         nil,
			inputFiles:     inputFiles[i],
			outputFile:     fmt.Sprintf("%s/%d", s.outputDir, i),
			state:          NotAssigned,
			pluginFilepath: s.pluginFilePath,
		})
		log.Info().Msgf("Created reduce task: %v", s.reduceTasks[i])
	}
	log.Info().Msg("Created reduce tasks")
	s.reduceTaskToProcess = len(s.reduceTasks)
	return nil
}

func (t *reduceTask) RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service) {
	log.Info().Msgf("Assigning reduce task %d to worker %v", t.id, worker)
	service.workers[worker] = false
	t.worker = worker
	t.state = Assigned
	inputFiles := t.inputFiles
	outputFile := t.outputFile
	go func() {
		res, err := (*worker).Reduce(ctx, &workerpb.ReduceRequest{
			IntermediateFiles: inputFiles,
			OutputFile:        outputFile,
			TaskId:            int32(t.id),
			PluginPath:        t.pluginFilepath,
		})
		if err != nil {
			log.Err(err).Msgf("ROUTINE: Error while processing reduce task: %v", err)
		}
		service.reduceResults <- res
		log.Info().Msgf("ROUTINE: Worker completed reduce task %d", t.id)
	}()
}

func (t *mapTask) RunTask(ctx context.Context, worker *workerpb.WorkerClient, service *Service) {
	log.Info().Msgf("Assigning map task %d to worker %v", t.id, worker)
	service.workers[worker] = false
	t.worker = worker
	t.ChangeState(Assigned)
	inputFile := t.inputFile
	intermediateDir := t.intermediateDir
	go func() {
		res, err := (*worker).Map(ctx, &workerpb.MapRequest{
			InputFile:       inputFile,
			IntermediateDir: intermediateDir,
			NumPartitions:   service.numberOfPartitions,
			TaskId:          int32(t.id),
			PluginPath:      t.pluginFilepath,
		})
		if err != nil {
			log.Err(err).Msgf("ROUTINE: Error while processing map task: %v", err)
		}
		service.mapResults <- res
		log.Info().Msgf("ROUTINE: Worker completed map task %d", t.id)
	}()
}

func checkWorkerStatus(ctx context.Context, worker *workerpb.WorkerClient, taskId int) WorkerState {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	res, err := (*worker).CheckStatus(ctxWithTimeout, &workerpb.CheckStatusRequest{})
	if err != nil {
		log.Err(err).Msgf("Error while checking worker status, for task_id %d: %v", taskId, err)
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
	task := s.mapTasks[idx]
	newId := len(s.mapTasks)
	taskDir := fmt.Sprintf("%s/map_%d", s.intermediateDir, newId)
	err := os.MkdirAll(taskDir, os.ModePerm)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to create intermediate directory: %v", err)
	}
	newTask := &mapTask{
		id:              newId,
		worker:          nil,
		inputFile:       task.inputFile,
		intermediateDir: taskDir,
		state:           NotAssigned,
	}
	s.mapTasks = append(s.mapTasks, newTask)
	return nil
}

func (s *Service) createReduceTaskCopy(idx int) error {
	task := s.reduceTasks[idx]
	newId := len(s.reduceTasks)
	newTask := &reduceTask{
		id:         newId,
		worker:     nil,
		inputFiles: task.inputFiles,
		outputFile: fmt.Sprintf("%s/%d", s.outputDir, newId),
		state:      NotAssigned,
	}
	s.reduceTasks = append(s.reduceTasks, newTask)
	return nil
}

func (s *Service) monitorTaskExecution(ctx context.Context, idx int, task RunnableTask) error {
	worker := task.CheckWorker()
	workerStatus := checkWorkerStatus(ctx, worker, task.CheckId())

	switch workerStatus {
	case Failure:
		// Worker is not processing the task - reassign the task
		task.ChangeState(Invalid)
		switch task.(type) {
		case *mapTask:
			err := s.createMapTaskCopy(idx)
			if err != nil {
				log.Err(err).Msgf("Monitor map task execution error while creating map task copy: %v", err)
				return err
			}
		case *reduceTask:
			err := s.createReduceTaskCopy(idx)
			if err != nil {
				log.Err(err).Msgf("Monitor map task execution error while creating reduce task copy: %v", err)
				return err
			}
		}
	case Idle, Processing:
		//	Worker is processing the task - come back later
	case Done:
		log.Info().Msgf("Master noticed that worker has completed task %d", task.CheckId())
		task.ChangeState(Completed)
		switch task.(type) {
		case *mapTask:
			s.mapTaskToProcess--
		case *reduceTask:
			s.reduceTaskToProcess--
		}
		s.mx.Lock()
		s.workers[task.CheckWorker()] = true
		s.mx.Unlock()
	default:
		panic("unhandled default case")
	}
	return nil
}

func (s *Service) assignTaskToWorker(ctx context.Context, task RunnableTask) {
	for worker := range s.workers {
		s.mx.Lock()
		available := s.workers[worker]
		s.mx.Unlock()
		if available {
			task.RunTask(ctx, worker, s)
			break
		}
	}
}

func (s *Service) processReduceTasks(ctx context.Context) error {
	log.Info().Msg("Processing reduce tasks")
	for s.reduceTaskToProcess > 0 {
		numberOfTasks := len(s.reduceTasks)
		for idx := range numberOfTasks {
			task := s.reduceTasks[idx]
			if task.CheckState() == NotAssigned {
				s.assignTaskToWorker(ctx, task)
			} else if task.CheckState() == Assigned {
				err := s.monitorTaskExecution(ctx, idx, task)
				if err != nil {
					return status.Errorf(codes.Internal, "Process reduce error while monitoring reduce task execution: %v", err)
				}
			}
		}
	}
	log.Info().Msg("Completed processing reduce tasks")
	return nil
}

func (s *Service) processMapTasks(ctx context.Context) error {
	log.Info().Msg("Processing map tasks")
	for s.mapTaskToProcess > 0 {
		numberOfTasks := len(s.mapTasks)
		for idx := range numberOfTasks {
			task := s.mapTasks[idx]
			if task.CheckState() == NotAssigned {
				s.assignTaskToWorker(ctx, task)
			} else if task.CheckState() == Assigned {
				err := s.monitorTaskExecution(ctx, idx, task)
				if err != nil {
					return status.Errorf(codes.Internal, "Process map error while monitoring map task execution: %v", err)
				}
			}
		}
	}
	log.Info().Msg("Completed processing map tasks")
	return nil
}

func NewService() *Service {
	return &Service{
		workers: make(map[*workerpb.WorkerClient]bool),
	}
}

func (s *Service) cleanup() error {
	log.Info().Msg("Cleaning up")
	err := os.RemoveAll(s.intermediateDir)
	if err != nil {
		log.Err(err).Msgf("Failed to remove intermediate directory: %v", err)
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

func (s *Service) initializeMapReduce(req *masterpb.MapReduceRequest) error {
	s.numberOfPartitions = req.GetNumPartitions()
	if s.numberOfPartitions <= 0 {
		return status.Errorf(codes.InvalidArgument, "Number of partitions should be greater than 0")
	}
	s.inputDir = req.GetInputDir()
	s.intermediateDir = req.GetWorkingDir() + intermediateDirName
	s.outputDir = req.GetWorkingDir() + outputDirName
	s.reduceResults = make(chan *workerpb.ReduceResponse, s.numberOfPartitions)
	s.pluginFilePath = req.GetPluginPath()
	return nil
}

func (s *Service) RegisterWorker(ctx context.Context, req *masterpb.RegisterWorkerRequest) (*masterpb.RegisterWorkerResponse, error) {
	log.Info().Msgf("Received RegisterWorkers request: %v", req)
	address := req.GetWorkerAddress().GetIp() + ":" + req.GetWorkerAddress().GetPort()
	s.mx.Lock()
	addressAlreadyExists := s.registeredAddress[address]
	s.mx.Unlock()
	if addressAlreadyExists {
		log.Info().Msgf("Worker already registered, no need to create new connection: %v", req.GetWorkerAddress())
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
	log.Info().Msgf("Registered successfully worker: %v", req.GetWorkerAddress())
	return &masterpb.RegisterWorkerResponse{}, nil
}

func (s *Service) MapReduce(ctx context.Context, req *masterpb.MapReduceRequest) (*masterpb.MapReduceResponse, error) {
	log.Info().Msgf("Received MapReduce request: %v", req)
	s.mr.Lock()
	defer s.mr.Unlock()
	logLineSeparator()
	err := s.initializeMapReduce(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to initialize map reduce: %v", err)
	}

	logLineSeparator()
	err = s.createMapTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create map tasks: %v", err)
	}
	logLineSeparator()
	err = s.processMapTasks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to process map tasks: %v", err)
	}

	logLineSeparator()
	err = s.createReduceTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create reduce tasks: %v", err)
	}

	logLineSeparator()
	err = s.processReduceTasks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to process reduce tasks: %v", err)
	}

	logLineSeparator()
	res, err := s.prepareResponse()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to prepare response: %v", err)
	}

	logLineSeparator()
	err = s.cleanup()
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Map reduce job completed")
	logLineSeparator()
	return res, nil
}
