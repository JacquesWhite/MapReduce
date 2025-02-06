package worker_startup

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/JacquesWhite/MapReduce/worker"
	"github.com/rs/zerolog/log"

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
		log.Fatal().Err(err).Msg("Failed to listen on Worker port")
	}

	srv := grpc.NewServer()
	ws := worker.NewWorkerService(ctx.MapFunc, ctx.ReduceFunc)
	workerpb.RegisterWorkerServer(srv, ws)
	reflection.Register(srv)

	serverError := make(chan error)

	// Handle the messages and do the work
	go func() {
		log.Info().Msgf("Worker is running on port %s", ctx.WorkerPort)
		serverError <- srv.Serve(listener)
	}()

	// Register Worker with Master on created ServiceWorker
	go func() {
		log.Info().Msg("Registering Worker with Master")
		serverError <- sendRegisterRequest(ctx)
	}()

	// Handle any errors that may occur on handled channel
	for {
		if err := <-serverError; err != nil {
			log.Fatal().Err(err).Msg("Error occurred on Worker server")
		}
	}
}

func sendRegisterRequest(ctx ContextWorker) error {
	// Send message to Master with Worker address for further communication.
	addr := ctx.MasterIP + ":" + ctx.MasterPort
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Master")
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
	log.Info().Msg("Sent RegisterWorker request with Worker info to Master")
	return err
}
