package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Sensor1Server struct {
	pb.UnimplementedSensorServiceServer
	edgeClient pb.EdgeServiceClient
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

func sendDataToEdgeService(client pb.EdgeServiceClient) {
	for {
		data := &pb.SensorData{
			Id:        1,
			Type:      "example_type",
			Value:     "example_value",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		edgeReq := &pb.EdgeRequest{Data: data}
		_, err := client.ProcessData(context.Background(), edgeReq)
		if err != nil {
			log.Printf("Error calling EdgeService: %v", err)
		} else {
			log.Printf("Sent data to EdgeService: %v", data)
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	grpcServer := grpc.NewServer()

	// Dial EdgeService
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to EdgeService: %v", err)
	}
	defer conn.Close()

	edgeClient := pb.NewEdgeServiceClient(conn)

	sensorServer := &Sensor1Server{edgeClient: edgeClient}
	pb.RegisterSensorServiceServer(grpcServer, sensorServer)

	go sendDataToEdgeService(edgeClient)

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
