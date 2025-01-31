package worker_startup

import (
	"context"
	"github.com/JacquesWhite/MapReduce/worker"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

type ContextWorker struct {
	MasterIP   string
	MasterPort string
	WorkerIP   string
	WorkerPort string
	MapFunc    worker.MapFuncT
	ReduceFunc worker.ReduceFuncT
}

func StartWorkerServer(ctx ContextWorker) {
	// Start server first, as it takes a second to boot up
	listener, err := net.Listen("tcp", ":"+ctx.WorkerPort)
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	ws := worker.NewWorkerService(ctx.MapFunc, ctx.ReduceFunc)
	workerpb.RegisterWorkerServer(srv, ws)
	reflection.Register(srv)

	serverError := make(chan error)

	// Register Worker with Master on created ServiceWorker
	go func() {
		// Handle the messages and do the work
		log.Println("Worker is running on port", ctx.WorkerPort)
		serverError <- srv.Serve(listener)
	}()

	go func() {
		log.Println("Registering Worker with Master")
		serverError <- sendRegisterRequest(ctx)
	}()

	// Wait for the Registration to fully be delivered
	for {
		if err := <-serverError; err != nil {
			log.Fatalf("Failed to register Worker with Master: %v", err)
		}
	}
}

func sendRegisterRequest(ctx ContextWorker) error {
	// Send message to Master with Worker address for further communication.
	addr := ctx.MasterIP + ":" + ctx.MasterPort
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
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
	log.Println("Sent RegisterWorker request with Worker info to Master")
	return err
}
