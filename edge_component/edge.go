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
	sensor1Client  pb.SensorServiceClient
	streamCtx      context.Context
	streamCancel   context.CancelFunc
	cloudClient    pb.CloudServiceClient
	cloudConn      *grpc.ClientConn
	cloudConnMutex sync.Mutex
	queue          chan *pb.StreamDataResponse
	queueCond      *sync.Cond
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
		// Wait for cloud connection to be established
		s.cloudConnMutex.Lock()
		for s.cloudConn == nil {
			s.cloudConnMutex.Unlock()
			s.queueCond.L.Lock()
			s.queueCond.Wait()
			s.queueCond.L.Unlock()
			s.cloudConnMutex.Lock()
		}
		s.cloudConnMutex.Unlock()

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
		stream, err = s.sensor1Client.StreamData(s.streamCtx, &pb.StreamDataRequest{})
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
		s.queue <- resp
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

func main() {
	sensor1Conn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to sensor 1: %v", err)
	}
	defer sensor1Conn.Close()
	sensor1Client := pb.NewSensorServiceClient(sensor1Conn)
	streamCtx, streamCancel := context.WithCancel(context.Background())

	edgeServer := &EdgeServer{
		sensor1Client: sensor1Client,
		streamCtx:     streamCtx,
		streamCancel:  streamCancel,
		queue:         make(chan *pb.StreamDataResponse, 100),
		queueCond:     sync.NewCond(&sync.Mutex{}),
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEdgeServiceServer(grpcServer, edgeServer)

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	reflection.Register(grpcServer)

	go edgeServer.handleStream()
	go edgeServer.connectToCloud()
	go edgeServer.processQueue()

	log.Printf("Starting gRPC server on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
