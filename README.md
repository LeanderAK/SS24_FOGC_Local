# Fog Computing Assignment

## Pre-requisites
- make
- protoc https://grpc.io/docs/protoc-installation/
- pip
- go 1.22.4 https://go.dev/doc/install
- protoc-gen-go and protoc-gen-go-grpc:
    ```bash
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
    ```
- python proto compilers
    ```bash
    python3 -m venv .venv
    source .venv/bin/activate
    pip install -r requirements.txt
    ```
- grpcui (optional for testing):
    ```bash
    go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
    ```

## Local Setup
Run:
```bash
make cloud
```
or 
```bash
make edge
```
or 
```bash
make sensor1
```
.

This will compile protos and start a gRPC development server locally.

Test with grpcui:
```bash
grpcui -plaintext localhost:50051
```
(change port depending on the service you want to test).