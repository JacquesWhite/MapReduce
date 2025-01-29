package master

import (
	"context"
	"fmt"
	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"log"
	"os"
)

const (
	intermediateDirName = "/intermediate"
	outputDirName       = "/output"
	// TODO: Change this, so that it is not hardcoded
	numberOfPartitions = 2
)

type TaskState int

const (
	NotAssigned TaskState = iota
	Assigned
	Completed
)

type MapTask struct {
	id              int
	worker          *workerpb.WorkerClient
	inputFile       string
	intermediateDir string
	state           TaskState
}

type ReduceTask struct {
	id         int
	worker     *workerpb.WorkerClient
	inputFiles []string
	outputFile string
	state      TaskState
}

type Service struct {
	masterpb.UnimplementedMasterServer

	numberOfPartitions int32

	// Maps worker client to a boolean value, which is true if the worker is available
	// and false when was busy when we last checked up
	workers map[*workerpb.WorkerClient]bool

	inputDir        string
	intermediateDir string
	outputDir       string

	// Number of map tasks that are left to be processed
	mapTaskToProcess int
	mapTasks         []*MapTask
	mapResults       chan *workerpb.MapResponse
	//completedMapTasks map[int]MapTask

	reduceTaskToProcess int
	reduceTasks         []*ReduceTask
	reduceResults       chan *workerpb.ReduceResponse
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

	for i, file := range files {
		taskDir := fmt.Sprintf("%s/map_%d/", s.intermediateDir, i)
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
	}
	s.mapTaskToProcess = len(s.mapTasks)
	s.mapResults = make(chan *workerpb.MapResponse, len(s.mapTasks))
	return nil
}

func (s *Service) processMapTasks(ctx context.Context) {
	for s.mapTaskToProcess > 0 {
		for _, task := range s.mapTasks {
			if task.state == NotAssigned {
				for worker, available := range s.workers {
					if available {
						s.workers[worker] = false
						task.worker = worker
						task.state = Assigned
						go func() {
							res, err := (*worker).Map(ctx, &workerpb.MapRequest{
								InputFile:       task.inputFile,
								IntermediateDir: task.intermediateDir,
								NumPartitions:   s.numberOfPartitions,
							})
							if err != nil {
								log.Printf("Error while processing map task: %v", err)
							}
							task.state = Completed
							s.mapTaskToProcess--
							s.mapResults <- res
							s.workers[worker] = true
						}()
						break
					}
				}
			}
		}
	}
}

func (s *Service) createReduceTasks() error {
	if s.mapTaskToProcess != 0 {
		return status.Errorf(codes.Internal, "All map tasks should have been processed before creating reduce tasks")
	}
	for i := 0; i < int(s.numberOfPartitions); i++ {
		inputFiles := make([]string, 0)
		for _, task := range s.mapTasks {
			inputFiles = append(inputFiles, fmt.Sprintf("%s/%d", task.intermediateDir, i))
		}
		s.reduceTasks = append(s.reduceTasks, &ReduceTask{
			id:         i,
			worker:     nil,
			inputFiles: inputFiles,
			outputFile: fmt.Sprintf("%s/%d", s.outputDir, i),
			state:      NotAssigned,
		})
	}
	s.reduceTaskToProcess = len(s.reduceTasks)
	return nil
}

func (s *Service) processReduceTasks(ctx context.Context) {
	for s.reduceTaskToProcess > 0 {
		for _, task := range s.reduceTasks {
			for worker, available := range s.workers {
				if available {
					s.workers[worker] = false
					task.worker = worker
					task.state = Assigned
					go func() {
						res, err := (*worker).Reduce(ctx, &workerpb.ReduceRequest{
							IntermediateFiles: task.inputFiles,
							OutputFile:        task.outputFile,
						})
						if err != nil {
							log.Printf("Error while processing reduce task: %v", err)
						}
						task.state = Completed
						s.workers[worker] = true
						s.reduceResults <- res
						s.reduceTaskToProcess--
					}()
					break
				}
			}
		}
	}
}

func NewService() *Service {
	return &Service{
		workers: make(map[*workerpb.WorkerClient]bool),
	}
}

func (s *Service) RegisterWorker(ctx context.Context, req *masterpb.RegisterWorkerRequest) (*masterpb.RegisterWorkerResponse, error) {
	log.Printf("Received RegisterWorkers request: %v", req)
	workerClient, err := createWorkerClient(req.GetWorkerAddress())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: failed to create worker client: %v", err)
	}
	// Check if worker can accept connections
	res, err := workerClient.CheckStatus(ctx, &workerpb.CheckStatusRequest{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: failed to check worker status: %v", err)
	}
	if res.GetStatus() != workerpb.CheckStatusResponse_IDLE {
		return nil, status.Errorf(codes.Internal, "RegisterWorker: worker is not ready to accept connections")
	}

	s.workers[&workerClient] = true
	return &masterpb.RegisterWorkerResponse{}, nil
}

func (s *Service) MapReduce(ctx context.Context, req *masterpb.MapReduceRequest) (*masterpb.MapReduceResponse, error) {
	log.Printf("Received MapReduce request: %v", req)
	s.inputDir = req.GetInputDir()
	s.intermediateDir = req.GetWorkingDir() + intermediateDirName
	s.outputDir = req.GetWorkingDir() + outputDirName
	s.numberOfPartitions = numberOfPartitions
	s.reduceResults = make(chan *workerpb.ReduceResponse, s.numberOfPartitions)

	err := s.createMapTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create map tasks: %v", err)
	}
	s.processMapTasks(ctx)

	err = s.createReduceTasks()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Map reduce: failed to create reduce tasks: %v", err)
	}
	s.processReduceTasks(ctx)

	log.Printf("Completed map reduce job")
	return &masterpb.MapReduceResponse{
		OutputDir: s.outputDir,
	}, nil
}
