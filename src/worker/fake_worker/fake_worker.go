package fake_worker

import (
	"context"
	"log"
	"sync"
	"time"

	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

// Service is a fake worker service, used for presenting master error handling capabilities
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

func (s *Service) CheckStatus(_ context.Context, _ *workerpb.CheckStatusRequest) (*workerpb.CheckStatusResponse, error) {
	// While executing a task, worker has the mutex locked, so we won't be able to get the status
	s.mx.Lock()
	res := &workerpb.CheckStatusResponse{Status: s.status}
	s.mx.Unlock()
	return res, nil
}

func (s *Service) Map(_ context.Context, req *workerpb.MapRequest) (*workerpb.MapResponse, error) {
	log.Printf("Received Map request: %v", req)
	// Implement your map logic here
	s.mx.Lock()
	// Master should mark this worker as "not responding" withing 10 seconds
	time.Sleep(10 * time.Second)
	s.mx.Unlock()
	log.Printf("Map task completed")
	return &workerpb.MapResponse{}, nil
}

func (s *Service) Reduce(_ context.Context, req *workerpb.ReduceRequest) (*workerpb.ReduceResponse, error) {
	// Implement your map logic here
	log.Printf("Received Reduce request: %v", req)
	s.mx.Lock()
	// Master should mark this worker as "not responding" withing 10 seconds
	time.Sleep(10 * time.Second)
	s.mx.Unlock()
	log.Printf("Reduce task completed")
	return &workerpb.ReduceResponse{}, nil
}
