package main

import (
	"context"
	"log"
	"net"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Sensor1Server struct {
	pb.UnimplementedSensorServiceServer
}

func (s *Sensor1Server) GetData(ctx context.Context, req *pb.SensorRequest) (*pb.SensorResponse, error) {
	log.Printf("GetData request: %d", req.Id)
	data := &pb.SensorData{
		Id:        req.Id,
		Type:      "example_type",
		Value:     "example_value",
		Timestamp: "example_timestamp",
	}
	return &pb.SensorResponse{Data: data}, nil
}

func main() {
	grpcServer := grpc.NewServer()

	pb.RegisterSensorServiceServer(grpcServer, &Sensor1Server{})

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
