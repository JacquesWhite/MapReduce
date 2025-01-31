package main

import (
	"context"
	"flag"
	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/JacquesWhite/MapReduce/worker/fake_worker"

	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

// This should be only used for presenting master error handling capabilities
func main() {
	workerPort := flag.String("w_port", "50001", "The worker port")
	workerIP := flag.String("w_ip", "localhost", "The worker IP")
	masterPort := flag.String("m_port", "50051", "The master port")
	masterIP := flag.String("m_ip", "localhost", "The master IP")
	flag.Parse()

	listener, err := net.Listen("tcp", ":"+*workerPort)
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	service := fake_worker.NewService()
	workerpb.RegisterWorkerServer(srv, service)
	reflection.Register(srv)

	serverError := make(chan error)

	go func() {
		log.Println("Worker is running on port", *workerPort)
		serverError <- srv.Serve(listener)
	}()

	// Register Worker with Master on created ServiceWorker
	go func() {
		log.Println("Registering Worker with Master")
		serverError <- sendRegisterRequest(*masterIP, *masterPort, *workerIP, *workerPort)
	}()

	// Handle any errors that may occur on handled channel
	for {
		if err := <-serverError; err != nil {
			log.Fatalf("Failed to register Worker with Master: %v", err)
		}
	}
}

func sendRegisterRequest(masterIP, masterPort, workerIP, workerPort string) error {
	addr := masterIP + ":" + masterPort
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := masterpb.NewMasterClient(conn)
	req := &masterpb.RegisterWorkerRequest{
		WorkerAddress: &masterpb.WorkerAddress{
			Ip:   workerIP,
			Port: workerPort,
		},
	}

	_, err = client.RegisterWorker(context.Background(), req)
	return err
}
