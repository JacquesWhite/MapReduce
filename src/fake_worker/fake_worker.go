package fake_worker

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

type Service struct {
	workerpb.UnimplementedWorkerServer
	mx sync.Mutex

	status workerpb.CheckStatusResponse_Status
}

func NewService() *Service {
	return &Service{
		status: workerpb.CheckStatusResponse_IDLE,
	}
}

func (s *Service) Init(masterAddress string, port string) {
	log.Printf("Connecting to master at %s", masterAddress)
	log.Printf("Worker is listening on port %s", port)
	conn, err := grpc.NewClient(masterAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	masterClient := masterpb.NewMasterClient(conn)
	_, err = masterClient.RegisterWorker(context.Background(), &masterpb.RegisterWorkerRequest{
		WorkerAddress: &masterpb.WorkerAddress{
			Ip:   "localhost",
			Port: port,
		},
	})
	if err != nil {
		panic(err)
	}
}

func (s *Service) CheckStatus(_ context.Context, _ *workerpb.CheckStatusRequest) (*workerpb.CheckStatusResponse, error) {
	s.mx.Lock()
	res := &workerpb.CheckStatusResponse{Status: s.status}
	s.mx.Unlock()
	return res, nil
}

func (s *Service) Map(_ context.Context, req *workerpb.MapRequest) (*workerpb.MapResponse, error) {
	log.Printf("Received Map request: %v", req)
	// Implement your map logic here
	s.mx.Lock()
	s.status = workerpb.CheckStatusResponse_BUSY
	s.mx.Unlock()

	// Check if the input file exists
	log.Printf("Checking if input file %s exists", req.InputFile)
	if _, err := os.Stat(req.InputFile); os.IsNotExist(err) {
		return nil, status.Errorf(codes.Internal, "input file %s does not exist", req.InputFile)
	}
	log.Print("OK")

	for i := 0; i < int(req.NumPartitions); i++ {
		fileName := filepath.Join(req.IntermediateDir, strconv.Itoa(i))
		file, err := os.Create(fileName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Map: create file failed %v", err)
		}
		err = file.Close()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Map: close file failed %v", err)
		}
	}
	s.mx.Lock()
	s.status = workerpb.CheckStatusResponse_IDLE
	s.mx.Unlock()
	log.Printf("Map task completed")
	return &workerpb.MapResponse{}, nil
}

func (s *Service) Reduce(_ context.Context, req *workerpb.ReduceRequest) (*workerpb.ReduceResponse, error) {
	// Implement your map logic here
	log.Printf("Received Reduce request: %v", req)
	s.mx.Lock()
	s.status = workerpb.CheckStatusResponse_BUSY
	s.mx.Unlock()

	// Check if the input file exists
	for _, inputFile := range req.IntermediateFiles {
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			return nil, status.Errorf(codes.Internal, "input file %s does not exist", inputFile)
		}
	}

	file, err := os.Create(req.OutputFile)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	s.mx.Lock()
	s.status = workerpb.CheckStatusResponse_IDLE
	s.mx.Unlock()
	log.Printf("Reduce task completed")
	return &workerpb.ReduceResponse{}, nil
}
