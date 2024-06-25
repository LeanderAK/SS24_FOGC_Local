package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type EdgeServer struct {
	pb.UnimplementedEdgeServiceServer
	sensor1Client    pb.SensorServiceClient
	sensor1Conn      *grpc.ClientConn
	streamCtx        context.Context
	streamCancel     context.CancelFunc
	sensor1ConnMutex sync.Mutex
	cloudClient      pb.CloudServiceClient
	cloudConn        *grpc.ClientConn
	cloudConnMutex   sync.Mutex
	queue            chan *pb.StreamDataResponse
	queueCond        *sync.Cond
}

func (s *EdgeServer) processStream(ctx context.Context, req *pb.StreamDataResponse) error {
	s.cloudConnMutex.Lock()
	defer s.cloudConnMutex.Unlock()

	if s.cloudConn == nil {
		return fmt.Errorf("cloud connection is down")
	}

	_, err := s.cloudClient.ProcessData(ctx, &pb.ProcessDataRequest{Data: req.Data})
	if err != nil {
		return err
	}

	log.Printf("Cloud response ok")
	return nil
}

func (s *EdgeServer) processQueue() {
	for msg := range s.queue {
		s.queueCond.L.Lock()
		for s.cloudConn == nil {
			s.queueCond.Wait()
		}
		s.queueCond.L.Unlock()

		err := s.processStream(context.Background(), msg)
		if err != nil {
			log.Printf("Failed to process queued data: %v. Requeuing data.", err)
			s.queue <- msg
		}
	}
}

func (s *EdgeServer) handleStream() {
	var stream pb.SensorService_StreamDataClient
	var err error
	for {
		if s.sensor1Client == nil {
			log.Println("Sensor 1 client not connected. Retrying in 1 second...")
			time.Sleep(1 * time.Second)
			continue
		}
		stream, err = s.sensor1Client.StreamData(s.streamCtx, &pb.StreamDataRequest{})
		if err != nil {
			log.Printf("Failed to start stream: %v. Retrying in 1 second...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Stream receive error: %v", err)
				break
			}
			s.queue <- resp
		}
	}
}

func (s *EdgeServer) UpdatePosition(ctx context.Context, req *pb.UpdatePositionRequest) (*pb.UpdatePositionResponse, error) {
	log.Printf("Received position update: %+v", req.Position)
	return &pb.UpdatePositionResponse{}, nil
}

func (s *EdgeServer) connectToCloud() {
	for {
		s.cloudConnMutex.Lock()
		if s.cloudConn == nil {
			log.Println("Connecting to cloud...")
			conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to cloud: %v. Retrying in 1 second...", err)
				s.cloudConnMutex.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			s.cloudConn = conn
			s.cloudClient = pb.NewCloudServiceClient(conn)
			s.queueCond.Broadcast()
		}
		s.cloudConnMutex.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func (s *EdgeServer) connectToSensor1() {
	for {
		s.sensor1ConnMutex.Lock()
		if s.sensor1Conn == nil {
			log.Println("Connecting to sensor 1...")
			conn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to sensor 1: %v. Retrying in 1 second...", err)
				s.sensor1ConnMutex.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			s.sensor1Conn = conn
			s.sensor1Client = pb.NewSensorServiceClient(conn)
			streamCtx, streamCancel := context.WithCancel(context.Background())
			s.streamCtx = streamCtx
			s.streamCancel = streamCancel

		}
		s.sensor1ConnMutex.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func main() {
	mu := sync.Mutex{}
	edgeServer := &EdgeServer{
		queue:     make(chan *pb.StreamDataResponse, 100),
		queueCond: sync.NewCond(&mu),
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEdgeServiceServer(grpcServer, edgeServer)

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	reflection.Register(grpcServer)

	go edgeServer.connectToSensor1()
	go edgeServer.connectToCloud()
	go edgeServer.handleStream()
	go edgeServer.processQueue()

	log.Printf("Starting gRPC server on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
