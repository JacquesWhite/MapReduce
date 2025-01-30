package main

import (
	"github.com/JacquesWhite/MapReduce/master"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
	"net"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
)

func main() {
	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	masterService := master.NewService()
	masterpb.RegisterMasterServer(grpcServer, masterService)

	// Enable server reflection
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	log.Println("Starting gRPC server on port 50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
