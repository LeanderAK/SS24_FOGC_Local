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

cloud: protogen
	python3 cloud_component/main.py

stop_edge:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50052) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

edge: protogen
	CLOUD_PORT=50051 EDGE_PORT=50052 SENSOR1_PORT=50053 SENSOR2_PORT=50054 go run edge/edge.go

stop_sensor1:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50053) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

sensor1: protogen
	SENSOR_ID=1 SENSOR_TYPE="velocity" SENSOR_PORT=50053 go run sensor/sensor.go

stop_sensor2:
	sudo /bin/sh -c 'pid=$$(lsof -t -i:50054) && [ -n "$$pid" ] && sudo kill -9 $$pid || true'

sensor2: protogen
	SENSOR_ID=2 SENSOR_TYPE="gyroscope" SENSOR_PORT=50054 go run sensor/sensor.go

stop: stop_cloud stop_edge stop_sensor1 stop_sensor2