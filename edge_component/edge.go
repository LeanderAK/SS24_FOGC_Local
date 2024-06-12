package main

import (
	"context"
	"log"
	"net"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type EdgeServer struct {
	pb.UnimplementedEdgeServiceServer
}

func (s *EdgeServer) ProcessData(ctx context.Context, req *pb.EdgeRequest) (*pb.EdgeResponse, error) {
	log.Printf("Processing data: %+v", req)
	return &pb.EdgeResponse{}, nil
}


func main() {
	grpcServer := grpc.NewServer()

	pb.RegisterEdgeServiceServer(grpcServer, &EdgeServer{})

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

    // Enable reflection (for grpcui and grpcurl)
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port 50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
