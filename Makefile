GOBIN := $(shell go env GOPATH)/bin

all: protogen

protogen: proto/fog_grpc.pb.go proto/fog.pb.go proto/fog_pb2.py proto/fog_pb2_grpc.py proto/fog_pb2.pyi proto/fog_pb2_grpc.pyi

proto/fog_grpc.pb.go proto/fog.pb.go proto/fog_pb2.py proto/fog_pb2_grpc.py proto/fog_pb2.pyi proto/fog_pb2_grpc.pyi:
	protoc -I=./proto --go_out=./proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative --plugin=protoc-gen-go=$(GOBIN)/protoc-gen-go --plugin=protoc-gen-go-grpc=$(GOBIN)/protoc-gen-go-grpc ./proto/*.proto
	python3 -m grpc_tools.protoc -I ./proto --python_out=./proto --grpc_python_out=./proto --mypy_out=./proto --mypy_grpc_out=./proto ./proto/*.proto 

protoclean:
	rm -f proto/fog_grpc.pb.go proto/fog.pb.go proto/fog_pb2.py proto/fog_pb2_grpc.py proto/fog_pb2.pyi proto/fog_pb2_grpc.pyi

stop_cloud:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50051) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

cloud: protogen stop_cloud
	python3 cloud_component/main.py

stop_edge:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50052) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

edge: protogen stop_edge
	go run edge_component/edge.go

stop_sensor1:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50053) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

sensor1: protogen stop_sensor1
	go run sensor_1/sensor1.go

stop: stop_cloud stop_edge stop_sensor1