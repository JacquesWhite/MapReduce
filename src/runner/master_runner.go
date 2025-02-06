package main

import (
	"flag"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/JacquesWhite/MapReduce/master"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	masterpb "github.com/JacquesWhite/MapReduce/proto/master"
)

func main() {
	// Define a flag for the port
	port := flag.String("port", "50051", "The server port")

	// Set logging settings
	zerolog.LevelFieldName = "severity"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Parse the flags
	flag.Parse()

	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	masterService := master.NewService()
	masterpb.RegisterMasterServer(grpcServer, masterService)

	// Enable server reflection
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to listen on port %s", *port)
	}

	log.Info().Msgf("Starting gRPC server on port %s", *port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve gRPC server")
	}
}
