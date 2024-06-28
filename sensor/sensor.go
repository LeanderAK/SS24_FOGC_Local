package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	pb "github.com/LeanderAK/SS24_FOGC_Project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Sensor1Server struct {
	pb.UnimplementedSensorServiceServer
	id         string
	sensorType string
}

func (s *Sensor1Server) StreamData(req *pb.StreamDataRequest, stream pb.SensorService_StreamDataServer) error {
	for {
		data, err := generateData(s.id, s.sensorType)
		if data == nil {
			return fmt.Errorf("Failed to generate data: %v", err)
		}
		if err := stream.Send(&pb.StreamDataResponse{Data: data}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func generateData(sensorId, sensorType string) (*pb.SensorData, error) {
	rand.Seed(time.Now().UnixNano())
	switch sensorType {
	case "velocity":
		return &pb.SensorData{
			SensorId:  sensorId,
			Type:      pb.SensorType_VELOCITY,
			Value:     fmt.Sprintf("%.2f", rand.Float64()*100),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}, nil
	case "gyroscope":
		return &pb.SensorData{
			SensorId:  sensorId,
			Type:      pb.SensorType_GYROSCOPE,
			Value:     fmt.Sprintf("%.2f", rand.Float64()*360),
			Timestamp: time.Now().UTC().Format(time.RFC1123),
		}, nil
	default:
		return nil, fmt.Errorf("Unknown sensor type: %s", sensorType)
	}
}

func getEnv() (sensorId, sensorType, sensorIp, sensorPort string, err error) {
	sensorId = os.Getenv("SENSOR_ID")
	if sensorId == "" {
		err := fmt.Errorf("SENSOR_ID not set")
		return "", "", "", "", err
	}
	sensorType = os.Getenv("SENSOR_TYPE")
	if sensorType == "" {
		err := fmt.Errorf("SENSOR_TYPE not set")
		return "", "", "", "", err
	}
	sensorIp = os.Getenv("SENSOR_IP")
	if sensorIp == "" {
		sensorIp = "0.0.0.0"
	}
	sensorPort = os.Getenv("SENSOR_PORT")
	if sensorPort == "" {
		sensorPort = "50053"
	}
	return sensorId, sensorType, sensorIp, sensorPort, nil
}

func main() {
	sensorId, sensorType, ip, port, err := getEnv()
	if err != nil {
		log.Fatalf("Failed to get environment variables: %v", err)
	}

	grpcServer := grpc.NewServer()

	sensorServer := &Sensor1Server{
		id:         sensorId,
		sensorType: sensorType,
	}
	pb.RegisterSensorServiceServer(grpcServer, sensorServer)

	address := net.JoinHostPort(ip, port)
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
