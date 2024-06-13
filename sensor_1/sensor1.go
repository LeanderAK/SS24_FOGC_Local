package main

import (
	"log"
	"net"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Sensor1Server struct {
	pb.UnimplementedSensorServiceServer
}

func (s *Sensor1Server) StreamData(req *pb.SensorRequest, stream pb.SensorService_StreamDataServer) error {
	for {
		data := &pb.SensorData{
			Id:        req.Id,
			Type:      "example_type",
			Value:     "example_value",
			Timestamp: "example_timestamp",
		}
		if err := stream.Send(&pb.SensorResponse{Data: data}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	grpcServer := grpc.NewServer()

	sensorServer := &Sensor1Server{}
	pb.RegisterSensorServiceServer(grpcServer, sensorServer)

	listener, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Enable reflection (for grpcui and grpcurl)
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port 50053")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
