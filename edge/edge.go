package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
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
	streamCtx    *context.Context
	streamCancel *context.CancelFunc
}

type Cloud struct {
	cloudClient pb.CloudServiceClient
	cloudConn   *grpc.ClientConn
	stream      pb.CloudService_ProcessDataStreamClient
}

type EdgeServer struct {
	pb.UnimplementedEdgeServiceServer

	queue     chan *pb.StreamDataResponse
	queueCond *sync.Cond

	sensor1      *Sensor
	sensor1Mutex sync.Mutex
	sensor2      *Sensor
	sensor2Mutex sync.Mutex

	cloud      *Cloud
	cloudMutex sync.Mutex
}

func (s *EdgeServer) handleSensorStream(sensor **Sensor, sensorMutex *sync.Mutex) {
	var stream pb.SensorService_StreamDataClient
	var err error
	for {
		sensorMutex.Lock()
		if *sensor == nil {
			sensorMutex.Unlock()
			log.Println("Sensor client not connected. ")
			time.Sleep(1 * time.Second)
			continue
		}

		client := *(*sensor).client
		sensorID := (*sensor).id

		ctx, cancel := context.WithCancel(context.Background())
		stream, err = client.StreamData(ctx, &pb.StreamDataRequest{})
		if err != nil {
			log.Printf("Failed to start stream with sensor %s: %v. Retrying in 1 second...", sensorID, err)
			cancel()
			sensorMutex.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Stream receive error: %v", err)
				cancel()
				sensorMutex.Unlock()
				break
			}
			s.queue <- resp
		}
	}
}

func (s *EdgeServer) handleCloudStream() {
	for {
		if s.cloud == nil || s.cloud.stream == nil {
			log.Println("Cloud stream not connected. ")
			time.Sleep(1 * time.Second)
			continue
		}
		resp, err := s.cloud.stream.Recv()
		if err != nil {
			log.Printf("Cloud stream receive error: %v", err)
			s.cloudMutex.Lock()
			s.cloud.stream.CloseSend()
			s.cloud.stream = nil
			s.cloud.cloudConn.Close()
			s.cloud.cloudConn = nil
			s.cloudMutex.Unlock()
			continue
		}
		log.Printf("Received position update: %s", resp.Position)
	}
}

func (s *EdgeServer) processStream(ctx context.Context, req *pb.StreamDataResponse) error {
	s.cloudMutex.Lock()
	defer s.cloudMutex.Unlock()

	if s.cloud.cloudConn == nil {
		return fmt.Errorf("cloud connection is down")
	}

	err := s.cloud.stream.Send(&pb.ProcessDataRequest{Data: req.Data})
	if err != nil {
		return err
	}

	return nil
}

func (s *EdgeServer) processQueue() {
	for msg := range s.queue {
		log.Printf("Queue length: %d", len(s.queue))
		s.queueCond.L.Lock()
		for s.cloud == nil || s.cloud.cloudConn == nil {
			s.queueCond.Wait()
		}
		s.queueCond.L.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := s.processStream(ctx, msg)
		cancel()
		if err != nil {
			log.Printf("Failed to process queued data: %v. Requeuing data and waiting 1 second.", err)
			s.queue <- msg
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (s *EdgeServer) establishCloudConnection(addr string) {
	s.cloudMutex = sync.Mutex{}
	for {
		s.cloudMutex.Lock()
		if s.cloud == nil || s.cloud.cloudConn == nil || s.cloud.stream == nil {
			log.Printf("Connecting to cloud (%s)...", addr)
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to cloud: %v. Retrying in 1 second...", err)
				s.cloudMutex.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			client := pb.NewCloudServiceClient(conn)
			stream, err := client.ProcessDataStream(context.Background())
			if err != nil {
				log.Printf("Failed to start stream with cloud: %v. Retrying in 1 second...", err)
				conn.Close()
				s.cloudMutex.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}

			s.cloud = &Cloud{
				cloudClient: client,
				cloudConn:   conn,
				stream:      stream,
			}
			s.queueCond.Broadcast()
		}
		s.cloudMutex.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func (s *EdgeServer) connectToSensor(id, addr string) (sensor *Sensor, err error) {
	log.Printf("Connecting to sensor %s...", id)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sensor %s: %v", id, err)
	}
	client := pb.NewSensorServiceClient(conn)
	return &Sensor{
		id:     id,
		client: &client,
	}, nil
}

func (s *EdgeServer) establishSensorConnection(sensor **Sensor, id, addr string) {
	for {
		if *sensor != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Attempting client connection with sensor id=%s", id)
		newSensor, err := s.connectToSensor(id, addr)
		if err != nil {
			log.Printf("Failed to connect to sensor %s: %v. Retrying in 1 second", id, err)
			time.Sleep(1 * time.Second)
			continue
		}
		*sensor = newSensor
	}
}

func getPortFromEnv(varName string) (uint16, error) {
	port64, err := strconv.ParseUint(os.Getenv(varName), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("error converting string to uint16 for %s: %v", varName, err)
	}
	return uint16(port64), nil

}

func getEnv() (edgeIP, edgePort, sensor1IP, sensor1Port, sensor2IP, sensor2Port, cloudIP, cloudPort string) {
	edgeIP = os.Getenv("EDGE_IP")
	edgePort = os.Getenv("EDGE_PORT")
	sensor1IP = os.Getenv("SENSOR1_IP")
	sensor1Port = os.Getenv("SENSOR1_PORT")
	sensor2IP = os.Getenv("SENSOR2_IP")
	sensor2Port = os.Getenv("SENSOR2_PORT")
	cloudIP = os.Getenv("CLOUD_IP")
	cloudPort = os.Getenv("CLOUD_PORT")
	if edgeIP == "" {
		edgeIP = "0.0.0.0"
	}
	if edgePort == "" {
		edgePort = "50052"
	}
	localhost := "localhost"
	if sensor1IP == "" {
		sensor1IP = localhost
	}
	if sensor2Port == "" {
		sensor2Port = "50053"
	}
	if sensor2IP == "" {
		sensor2IP = localhost
	}
	if sensor2Port == "" {
		sensor2Port = "50054"
	}
	if cloudIP == "" {
		cloudIP = localhost
	}
	if cloudPort == "" {
		cloudPort = "50051"
	}
	return edgeIP, edgePort, sensor1IP, sensor1Port, sensor2IP, sensor2Port, cloudIP, cloudPort
}

func main() {
	edgeIP, edgePort, sensor1IP, sensor1Port, sensor2IP, sensor2Port, cloudIP, cloudPort := getEnv()

	edgeServer := &EdgeServer{
		queue:     make(chan *pb.StreamDataResponse, 100),
		queueCond: sync.NewCond(&sync.Mutex{}),
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEdgeServiceServer(grpcServer, edgeServer)

	addr := net.JoinHostPort(edgeIP, edgePort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	reflection.Register(grpcServer)

	// Continuously attempt to connect to the sensors
	sensor1Addr := net.JoinHostPort(sensor1IP, sensor1Port)
	go edgeServer.establishSensorConnection(&edgeServer.sensor1, "1", sensor1Addr)
	sensor2Addr := net.JoinHostPort(sensor2IP, sensor2Port)
	go edgeServer.establishSensorConnection(&edgeServer.sensor2, "2", sensor2Addr)

	// Establish and handle sensor streams
	go edgeServer.handleSensorStream(&edgeServer.sensor1, &edgeServer.sensor1Mutex)
	go edgeServer.handleSensorStream(&edgeServer.sensor2, &edgeServer.sensor2Mutex)
	go edgeServer.processQueue()

	// Establish cloud connection
	cloudAddr := net.JoinHostPort(cloudIP, cloudPort)
	go edgeServer.establishCloudConnection(cloudAddr)
	go edgeServer.handleCloudStream()

	log.Printf("Starting gRPC server on %s", addr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
