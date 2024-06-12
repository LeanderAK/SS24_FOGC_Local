GOBIN := $(shell go env GOPATH)/bin

all: protogen

protogen:
	protoc -I=./proto --go_out=./proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative --plugin=protoc-gen-go=$(GOBIN)/protoc-gen-go --plugin=protoc-gen-go-grpc=$(GOBIN)/protoc-gen-go-grpc ./proto/*.proto
	python3 -m grpc_tools.protoc -I ./proto --python_out=./proto --grpc_python_out=./proto --mypy_out=./proto --mypy_grpc_out=./proto ./proto/*.proto 

cloud: protogen
	python3 cloud_component/main.py

edge: protogen
	go run edge_component/edge.go

sensor1: protogen
	go run sensor_1/sensor1.go