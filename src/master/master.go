package master

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

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
	numberOfPartitions  = 2
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

	numberOfPartitions  int32
	workers             map[*workerpb.WorkerClient]bool
	inputDir            string
	intermediateDir     string
	outputDir           string
	mapTaskToProcess    int
	mapTasks            []*MapTask
	mapResults          chan *workerpb.MapResponse
	reduceTaskToProcess int
	reduceTasks         []*ReduceTask
	reduceResults       chan *workerpb.ReduceResponse
	mx                  sync.Mutex
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
		log.Printf("Created map task: %v", s.mapTasks[i])
	}
	s.mapTaskToProcess = len(s.mapTasks)
	s.mapResults = make(chan *workerpb.MapResponse, len(s.mapTasks))
	log.Printf("Created map tasks")
	return nil
}

func (s *Service) processMapTasks(ctx context.Context) {
	log.Printf("Processing map tasks")
	for s.mapTaskToProcess > 0 {
		for idx := range s.mapTasks {
			s.mx.Lock()
			task := s.mapTasks[idx]
			if task.state == NotAssigned {
				s.mx.Unlock()
				for worker := range s.workers {
					s.mx.Lock()
					available := s.workers[worker]
					if available {
						log.Printf("Assigning map task %d to worker %v", task.id, worker)
						s.workers[worker] = false
						task.worker = worker
						task.state = Assigned
						inputFile := task.inputFile
						intermediateDir := task.intermediateDir
						s.mx.Unlock()
						go func() {
							_, err := (*worker).Map(ctx, &workerpb.MapRequest{
								InputFile:       inputFile,
								IntermediateDir: intermediateDir,
								NumPartitions:   s.numberOfPartitions,
							})
							if err != nil {
								log.Printf("Error while processing map task: %v", err)
							}
							log.Printf("Completed map task %d", task.id)
							s.mx.Lock()
							log.Printf("2Completed map task %d", task.id)
							task.state = Completed
							s.mapTaskToProcess--
							s.workers[worker] = true
							//s.mapResults <- res
							s.mx.Unlock()
						}()
						break
					} else {
						s.mx.Unlock()
					}
				}
			} else {
				s.mx.Unlock()
			}
		}
	}
	log.Printf("Completed processing map tasks")
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
		log.Printf("Created reduce task: %v", s.reduceTasks[i])
	}
	log.Printf("Created reduce tasks")
	s.reduceTaskToProcess = len(s.reduceTasks)
	return nil
}

func (s *Service) processReduceTasks(ctx context.Context) {
	log.Printf("Processing reduce tasks")
	for s.reduceTaskToProcess > 0 {
		for idx := range s.reduceTasks {
			s.mx.Lock()
			task := s.reduceTasks[idx]
			if task.state == NotAssigned {
				s.mx.Unlock()
				for worker := range s.workers {
					s.mx.Lock()
					available := s.workers[worker]
					if available {
						log.Printf("Assigning reduce task %d to worker %v", task.id, worker)
						s.workers[worker] = false
						task.worker = worker
						task.state = Assigned
						inputFiles := task.inputFiles
						outputFile := task.outputFile
						s.mx.Unlock()
						go func() {
							_, err := (*worker).Reduce(ctx, &workerpb.ReduceRequest{
								IntermediateFiles: inputFiles,
								OutputFile:        outputFile,
							})
							if err != nil {
								log.Printf("Error while processing reduce task: %v", err)
							}
							log.Printf("Completed reduce task %d", task.id)
							s.mx.Lock()
							log.Printf("2Completed reduce task %d", task.id)
							task.state = Completed
							s.workers[worker] = true
							s.reduceTaskToProcess--
							//s.reduceResults <- res
							s.mx.Unlock()
						}()
						break
					} else {
						s.mx.Unlock()
					}
				}
			} else {
				s.mx.Unlock()
			}
		}
	}
	log.Printf("Completed processing reduce tasks")
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
	err = s.cleanup()
	if err != nil {
		return nil, err
	}
	return &masterpb.MapReduceResponse{
		OutputDir: s.outputDir,
	}, nil
}
