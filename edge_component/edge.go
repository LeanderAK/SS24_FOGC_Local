package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type EdgeServer struct {
	pb.UnimplementedEdgeServiceServer
	sensor1Client pb.SensorServiceClient
	streamCtx     context.Context
	streamCancel  context.CancelFunc

	cloudClient pb.CloudServiceClient
}

func (s *EdgeServer) ProcessData(ctx context.Context, req *pb.SensorResponse) error {
	log.Printf("Processing data: %+v", req)

	resp, err := s.cloudClient.ProcessData(ctx, &pb.CloudRequest{Data: req.Data})
	if err != nil {
		return err
	}

	// Somehow aggregate data
	log.Printf("Cloud response: %+v", resp)

	return nil
}

func (s *EdgeServer) handleStream() {
	var stream pb.SensorService_StreamDataClient
	var err error
	for {
		stream, err = s.sensor1Client.StreamData(s.streamCtx, &pb.SensorRequest{})
		if err == nil {
			break
		}
		log.Printf("Failed to start stream: %v. Retrying in 1 second...", err)
		time.Sleep(1 * time.Second)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Stream receive error: %v", err)
			break
		}
		err = s.ProcessData(s.streamCtx, resp)
		if err != nil {
			log.Printf("Failed to process data: %v", err)
		}
	}
}

func main() {
	// Set up the connection to sensor1
	sensor1Conn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to sensor 1: %v", err)
	}
	defer sensor1Conn.Close()
	sensor1Client := pb.NewSensorServiceClient(sensor1Conn)
	streamCtx, streamCancel := context.WithCancel(context.Background())

	// Set up the connection to cloud
	cloudConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to cloud: %v", err)
	}
	defer cloudConn.Close()
	cloudClient := pb.NewCloudServiceClient(cloudConn)

	edgeServer := &EdgeServer{
		sensor1Client: sensor1Client,
		streamCtx:     streamCtx,
		streamCancel:  streamCancel,
		cloudClient:   cloudClient,
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEdgeServiceServer(grpcServer, edgeServer)

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Enable reflection (for grpcui and grpcurl)
	reflection.Register(grpcServer)

	// Start the stream handler in a separate goroutine
	go edgeServer.handleStream()

	// Server has no endpoints for now, maybe add some later if needed
	log.Printf("Starting gRPC server on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
