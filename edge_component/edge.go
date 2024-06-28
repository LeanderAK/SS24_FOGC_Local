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

type Sensor struct {
	id           string
	client       *pb.SensorServiceClient
	connMutex    sync.Mutex
	streamCtx    *context.Context
	streamCancel *context.CancelFunc
}

type EdgeServer struct {
	pb.UnimplementedEdgeServiceServer

	queue     chan *pb.StreamDataResponse
	queueCond *sync.Cond

	sensor1Client *Sensor
	sensor2Client *Sensor

	cloudClient            pb.CloudServiceClient
	cloudConn              *grpc.ClientConn
	cloudConnMutex         sync.Mutex
	cloudProcessDataCtx    context.Context
	cloudProcessDataCancel context.CancelFunc
}

func (s *EdgeServer) UpdatePosition(ctx context.Context, req *pb.UpdatePositionRequest) (*pb.UpdatePositionResponse, error) {
	log.Printf("Received position update: %+v", req.Position)
	return &pb.UpdatePositionResponse{}, nil
}

func (s *EdgeServer) handleSensorStream(sensor **Sensor) {
	var stream pb.SensorService_StreamDataClient
	var err error
	for {
		if *sensor == nil {
			log.Println("Sensor 1 client not connected. ")
			time.Sleep(1 * time.Second)
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		client := *(*sensor).client
		stream, err = client.StreamData(ctx, &pb.StreamDataRequest{})
		if err != nil {
			log.Printf("Failed to start stream with sensor %s: %v. Retrying in 1 second...", (*sensor).id, err)
			cancel()
			time.Sleep(1 * time.Second)
			continue
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Stream receive error: %v", err)
				cancel()
				break
			}
			s.queue <- resp
		}
	}
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

		s.cloudProcessDataCtx, s.cloudProcessDataCancel = context.WithTimeout(context.Background(), time.Second)
		err := s.processStream(s.cloudProcessDataCtx, msg)
		s.cloudProcessDataCancel()
		if err != nil {
			log.Printf("Failed to process queued data: %v. Requeuing data.", err)
			s.queue <- msg
		}
	}
}

func (s *EdgeServer) establishCloudConnection() {
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

func (s *EdgeServer) connectToSensor(id, ip string, port uint16) (sensor *Sensor, err error) {
	log.Printf("Connecting to sensor %s...", id)
	target := fmt.Sprintf("%s:%d", ip, port)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sensor %s: %v", id, err)
	}
	client := pb.NewSensorServiceClient(conn)
	return &Sensor{
		id:        id,
		client:    &client,
		connMutex: sync.Mutex{},
	}, nil
}

func (s *EdgeServer) establishSensorConnection(sensor **Sensor, id, ip string, port uint16) {
	for {
		if *sensor != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Attempting client connection with sensor id=%s", id)
		newSensor, err := s.connectToSensor(id, ip, port)
		if err != nil {
			log.Printf("Failed to connect to sensor %s: %v. Retrying in 1 second", id, err)
			time.Sleep(1 * time.Second)
			continue
		}
		*sensor = newSensor
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

	go edgeServer.establishSensorConnection(&edgeServer.sensor1Client, "1", "localhost", uint16(50053))
	go edgeServer.establishSensorConnection(&edgeServer.sensor2Client, "2", "localhost", uint16(50054))
	go edgeServer.handleSensorStream(&edgeServer.sensor1Client)
	go edgeServer.handleSensorStream(&edgeServer.sensor2Client)
	go edgeServer.establishCloudConnection()
	go edgeServer.processQueue()

	log.Printf("Starting gRPC server on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
