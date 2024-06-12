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
	cloudClient pb.CloudServiceClient
}

func (s *EdgeServer) ProcessData(ctx context.Context, req *pb.EdgeRequest) (*pb.EdgeResponse, error) {
	log.Printf("Processing data: %+v", req)

	// Forward the request to the CloudService
	cloudReq := &pb.CloudRequest{
		Data: req.Data,
	}

	cloudResp, err := s.cloudClient.ProcessData(ctx, cloudReq)
	if err != nil {
		return nil, err
	}

	log.Printf("Received response from CloudService: %+v", cloudResp)

	return &pb.EdgeResponse{}, nil
}

func main() {
	// Set up the connection to the CloudService
	cloudConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to CloudService: %v", err)
	}
	defer cloudConn.Close()

	cloudClient := pb.NewCloudServiceClient(cloudConn)

	grpcServer := grpc.NewServer()

	edgeServer := &EdgeServer{
		cloudClient: cloudClient,
	}

	pb.RegisterEdgeServiceServer(grpcServer, edgeServer)

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Enable reflection (for grpcui and grpcurl)
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
