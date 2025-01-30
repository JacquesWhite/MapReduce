package worker

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

type MapFuncT = func(string, string) []KeyValue
type ReduceFuncT = func(string, []string) string

type ContextWorker struct {
	MasterIP   string
	MasterPort string
	WorkerIP   string
	WorkerPort string
	MapFunc    MapFuncT
	ReduceFunc ReduceFuncT
}

func StartWorkerServer(ctx ContextWorker) {
	// Start server first, as it takes a second to boot up
	listener, err := net.Listen("tcp", ":"+ctx.WorkerPort)
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	ws := NewWorkerService(ctx.MapFunc, ctx.ReduceFunc)
	workerpb.RegisterWorkerServer(srv, ws)
	reflection.Register(srv)

	// Register Worker with Master on created ServiceWorker
	go func() { ws.sendRegisterRequest(ctx) }()

	// Wait for the Registration to fully be delivered
	time.Sleep(1 * time.Second)

	// Handle the messages and do the work
	log.Println("Worker is running on port", ctx.WorkerPort)
	if err := srv.Serve(listener); err != nil {
		panic(err)
	}
}

func (w *ServiceWorker) sendRegisterRequest(ctx ContextWorker) {
	// Send message to Master with Worker address for further communication.
	addr := ctx.MasterIP + ":" + ctx.MasterPort
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}

	client := masterpb.NewMasterClient(conn)
	req := &masterpb.RegisterWorkerRequest{
		WorkerAddress: &masterpb.WorkerAddress{
			// My Worker IP given for me on input to later send it to Master.
			Ip: ctx.WorkerIP,

			// Client port passed to Master
			Port: ctx.WorkerPort,
		},
	}

	_, err = client.RegisterWorker(context.Background(), req)
	if err != nil {
		return
	}

	log.Println("Sent RegisterWorker request with Worker info to Master")
}
