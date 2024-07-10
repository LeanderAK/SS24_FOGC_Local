# Fog Computing Assignment
## Demonstration Video
https://www.youtube.com/watch?v=dKCi2xfCVrA

## Documentation
[Documentation](documentation.pdf)

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
- terraform (optional for GCP setup)
- gcloud (optional for GCP setup)

## Local Setup
Run in separate terminals:
```bash
make cloud
```
```bash
make edge
```
```bash
make sensor1
```
```bash
make sensor2
```
This will compile protos and start a gRPC development server locally.

Test with grpcui:
```bash
grpcui -plaintext localhost:50051
```
(change port depending on the service you want to test).

## GCP Setup
1. Create a GCP project and enable the Compute Engine API.
2. Run `gcoud init` to authenticate with GCP.
3. Select the project you created in step 1 by running `gcloud config set project <project_id>`.
4. Run `gcloud auth application-default login` to authenticate with GCP.
5. Edit the `project` and `region` in [`terraform/main.tf`](terraform/main.tf) to your GCP project and region.

## Fog Setup
1. Make sure port `50052` on your machine is exposed to the public internet. You might have to setup port forwarding on your home router (if in doubt, check this guide: https://www.noip.com/support/knowledgebase/general-port-forwarding-guide)
2. Edit the `EDGE_IP=<add_your_ip>` in [`terraform/main.tf`](terraform/main.tf) to your public IP (https://whatsmyip.com/).
3. Deploy the edge server to GCP by running:
    ```bash
    cd terraform
    terraform init
    terraform apply
    cd ..
    ``` 
    Press `yes` when prompted.
4. Wait for the cloud server to be deployed. This might take a few minutes. 
5. After the server is deployed, you will see the public IP of the edge server in the output (`instance_ip = "xx.xx.xx.xx"`). Copy this IP.
6. Run sensors on your machine (in separate terminals):
    ```bash
    make sensor1
    ```
    ```bash
    make sensor2
    ```
7. Run the edge server on your machine (using the public IP from step 5):
    ```bash
    CLOUD_IP=xx.xx.xx.xx make edge
    ```

