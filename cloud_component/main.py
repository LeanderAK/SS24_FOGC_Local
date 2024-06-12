import os
import sys
from concurrent import futures
import grpc
from grpc_reflection.v1alpha import reflection

project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if project_root not in sys.path:
    sys.path.append(project_root)

import proto.fog_pb2 as fog_pb2  # noqa
import proto.fog_pb2_grpc as fog_pb2_grpc  # noqa


class CloudService(fog_pb2_grpc.CloudServiceServicer):
    def ProcessData(self, request, context):
        # Implement your logic here
        response_data = fog_pb2.CloudData(result=42)  # Example result
        return fog_pb2.CloudResponse(data=response_data)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    fog_pb2_grpc.add_CloudServiceServicer_to_server(CloudService(), server)

    # Enable reflection (for grpcui and grpcurl)
    SERVICE_NAMES = (
        fog_pb2.DESCRIPTOR.services_by_name["CloudService"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port("[::]:50051")
    server.start()
    print("Server started on port 50051")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
