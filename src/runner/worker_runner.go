package main

import (
	"github.com/JacquesWhite/MapReduce/fake_worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"time"

	workerpb "github.com/JacquesWhite/MapReduce/proto/worker"
)

func main() {
	args := os.Args
	masterAddress := args[1]
	port := args[2]
	log.Printf("Master address: %s", masterAddress)
	log.Printf("Port: %s", port)
	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	workerService := fake_worker.NewService()
	workerpb.RegisterWorkerServer(grpcServer, workerService)

	// Enable server reflection
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		workerService.Init(masterAddress, port)
	}()
	log.Println("Starting gRPC server on port ", port, "...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
