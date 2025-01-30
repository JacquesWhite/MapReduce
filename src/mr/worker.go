package mr

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	pbmaster "github.com/JacquesWhite/MapReduce/proto/master"
	pbworker "github.com/JacquesWhite/MapReduce/proto/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type WorkerContext struct {
	MasterIP   string
	MasterPort string
	MapFunc    MapFuncT
	ReduceFunc ReduceFuncT
}

type WorkerService struct {
	pbworker.UnimplementedWorkerServer

	mapFunc    MapFuncT
	reduceFunc ReduceFuncT
	statusMx   sync.Mutex
	status     pbworker.CheckStatusResponse_Status
}

func NewWorkerService(m MapFuncT, r ReduceFuncT) *WorkerService {
	return &WorkerService{
		mapFunc:    m,
		reduceFunc: r,
		status:     pbworker.CheckStatusResponse_IDLE,
	}
}

func WorkerMain(workerCtx WorkerContext) {
	myPort := strconv.Itoa(rand.Intn(60000))
	startWorkerServer(workerCtx, myPort)
}

func startWorkerServer(ctx WorkerContext, port string) {
	// Start server first, as it takes a second to boot up
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	ws := NewWorkerService(ctx.MapFunc, ctx.ReduceFunc)
	pbworker.RegisterWorkerServer(srv, ws)
	reflection.Register(srv)

	// Register Worker with Master on created WorkerService
	go func() { ws.sendRegisterRequest(ctx.MasterIP, ctx.MasterPort, port) }()

	// Wait for the Registration to fully be delivered
	time.Sleep(1 * time.Second)

	// Handle the messages and do the work
	log.Println("Worker is running on port", port)
	if err := srv.Serve(listener); err != nil {
		panic(err)
	}
}

// Send message to Master with Worker address for further communication.
func (w *WorkerService) sendRegisterRequest(ip string, port string, mp string) {
	addr := ip + ":" + port
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	log.Println("Sent RegisterWorker request with Worker info to Master")
}

func (w *WorkerService) Map(ctx context.Context, request *pbworker.MapRequest) (*pbworker.MapResponse, error) {
	log.Println("Map request received")
	// Worker receives the message MapRequest
	// With file to Map and directory in which we want to create file with intermediate results

	// upon receiving a message from the master
	// which will contain the file name
	// open it and pass the contents to the Map function

	panic("implement me")
}

func (w *WorkerService) Reduce(ctx context.Context, request *pbworker.ReduceRequest) (*pbworker.ReduceResponse, error) {
	log.Println("Reduce request received")
	// Worker receives the message ReduceRequest
	// With file containing intermediate results and directory in which we want to create file with final results
	// todo 2.

	panic("implement me")
}

func (w *WorkerService) CheckStatus(_ context.Context, _ *pbworker.CheckStatusRequest) (*pbworker.CheckStatusResponse, error) {
	log.Println("CheckStatus request received")

	w.statusMx.Lock()
	status := w.status
	w.statusMx.Unlock()

	res := &pbworker.CheckStatusResponse{
		Status: status,
	}

	return res, nil
}

//var intermediate []KeyValue
//content, err := os.ReadFile("../datasets/test.txt")
//if err != nil {
//return
//}
//
//kva := workerCtx.MapFunc("../datasets/test.txt", string(content))
//intermediate = append(intermediate, kva...)
//
//sort.Sort(ByKey(intermediate))
//
//i := 0
//for i < len(intermediate) {
//j := i + 1
//for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
//j++
//}
//var values []string
//for k := i; k < j; k++ {
//values = append(values, intermediate[k].Value)
//}
//output := workerCtx.ReduceFunc(intermediate[i].Key, values)
//// this is the correct format for each line of Reduce output.
//// please do not change it.
//fmt.Println(intermediate[i].Key, output)
//i = j
//}
