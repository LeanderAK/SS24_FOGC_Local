package main

import (
	"log"
	"net"
	"os"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Sensor1Server struct {
	pb.UnimplementedSensorServiceServer
}

func (s *Sensor1Server) StreamData(req *pb.StreamDataRequest, stream pb.SensorService_StreamDataServer) error {
	for {
		data := &pb.SensorData{
			Id:        req.Id,
			Type:      "example_type",
			Value:     "example_value",
			Timestamp: "example_timestamp",
		}
		if err := stream.Send(&pb.StreamDataResponse{Data: data}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	ip := os.Getenv("SERVER_IP")
	if ip == "" {
		ip = "0.0.0.0"
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "50053"
	}

	address := net.JoinHostPort(ip, port)

	grpcServer := grpc.NewServer()

	sensorServer := &Sensor1Server{}
	pb.RegisterSensorServiceServer(grpcServer, sensorServer)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Enable reflection (for grpcui and grpcurl)
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on %s", address)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
