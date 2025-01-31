package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/JacquesWhite/MapReduce/master"
  
	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
)

func main() {
	// Define a flag for the port
	port := flag.String("port", "50051", "The server port")

	// Parse the flags
	flag.Parse()

	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	masterService := master.NewService()
	masterpb.RegisterMasterServer(grpcServer, masterService)

	// Enable server reflection
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}

	log.Println("Starting gRPC server on port ", *port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
