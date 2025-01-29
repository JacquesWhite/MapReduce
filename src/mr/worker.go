package mr

import (
	"context"
	//"fmt"
	"math/rand"
	"net"
	//"os"
	//"sort"
	"strconv"

	pbmaster "github.com/JacquesWhite/MapReduce/proto/master"
	pbworker "github.com/JacquesWhite/MapReduce/proto/worker"
	"google.golang.org/grpc"
)

type WorkerContext struct {
	MasterIP   string
	MasterPort string
	MapFunc    func(string, string) []KeyValue
	ReduceFunc func(string, []string) string
}

type WorkerServer struct {
	pbworker.UnimplementedWorkerServer
}

// Send message to Master with Worker address for further communication.
func sendRegisterRequest(ip string, port string, mp string) {
	conn, err := grpc.Dial(ip+":"+port, grpc.WithInsecure())
	if err != nil {
		return
	}

	client := pbmaster.NewMasterClient(conn)
	req := &pbmaster.RegisterWorkerRequest{
		WorkerAddress: &pbmaster.WorkerAddress{
			// Currently localhost, further may be passed as an argument
			Ip: "127.0.0.1",

			// Client port passed to Master
			Port: mp,
		},
	}

	_, err = client.RegisterWorker(context.Background(), req)
	if err != nil {
		return
	}

	if conn.Close() != nil {
		return
	}
}

func startWorkerServer(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return
	}

	srv := grpc.NewServer()
	pbworker.RegisterWorkerServer(srv, &WorkerServer{})

	for {
		// Handle the messages and do the work
		if srv.Serve(listener) != nil {
			return
		}
	}
}

func WorkerMain(workerCtx WorkerContext) {

	myPort := strconv.Itoa(rand.Intn(60000))
	sendRegisterRequest(workerCtx.MasterIP, workerCtx.MasterPort, myPort)

	startWorkerServer(myPort)
}

func (w WorkerServer) Map(ctx context.Context, request *pbworker.MapRequest) (*pbworker.MapResponse, error) {
	//TODO implement me
	panic("implement me")
	// Worker receives the message MapRequest
	// With file to Map and directory in which we want to create file with intermediate results

	// upon receiving a message from the master
	// which will contain the file name
	// open it and pass the contents to the Map function
}

func (w WorkerServer) Reduce(ctx context.Context, request *pbworker.ReduceRequest) (*pbworker.ReduceResponse, error) {
	//TODO implement me
	panic("implement me")
	// Worker receives the message ReduceRequest
	// With file containing intermediate results and directory in which we want to create file with final results
	// todo 2.
}

func (w WorkerServer) CheckStatus(ctx context.Context, request *pbworker.CheckStatusRequest) (*pbworker.CheckStatusResponse, error) {
	//TODO implement me
	panic("implement me")
}
