import os
import sys
from concurrent import futures
import time
import grpc
from grpc_reflection.v1alpha import reflection
import queue
import threading
from uuid import uuid4

project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if project_root not in sys.path:
    sys.path.append(project_root)

import proto.fog_pb2 as fog_pb2  # noqa
import proto.fog_pb2_grpc as fog_pb2_grpc  # noqa

class Task:
    def __init__(self, sensor_data):
        self.id = uuid4()
        self.value = sensor_data.value
        self.result = None
        self.processed = False

    def __str__(self):
        return f'Task_id: {self.id}, processed: {self.processed}'
    
class TaskQueue:
    def __init__(self):
        self.tasks = queue.Queue()
        self.lock = threading.Lock()

    def add_task(self, task: Task):
        with self.lock:
            self.tasks.put(task)

    def get_task(self) -> Task:
        with self.lock:
            if not self.tasks.empty():
                task = self.tasks.get()
                return task
            else:
                return None
            
class CloudService(fog_pb2_grpc.CloudServiceServicer):
    def __init__(self):
        self.task_queue = TaskQueue()
        self.feedback_url = "placeholder_url"

    def ProcessData(self, request: fog_pb2.ProcessDataRequest, context):
        response_data = fog_pb2.Position(x=10.0, y=20.0, z=0.5)
        print(f"Received data: {request.data}")
        print(type(request.data))
        if not request.ListFields():
            print("Data empty!")
        task = Task(request.data)
        self.task_queue.add_task(task=task)
        
        return fog_pb2.ProcessDataResponse()

    def process_task_queue(self):
        while True:
            task = self.task_queue.get_task()
            if task:
                if task.processed == False:
                    self.process_task(task=task)
                self.send_feedback(task=task)
            else:
                time.sleep(1)

    def process_task(self, task: Task):
        task_value = task.value
        
        # compute task result
        task.result = task_value + " SOME OPERATION"
        task.processed = True
        return task
    
    def send_feedback(self, task):
        data = {
            "result" : task.result
        }
        # Send data back to feedback_url
        # post(self.feedback_url, json=data)
    


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    cloud_service = CloudService()
    fog_pb2_grpc.add_CloudServiceServicer_to_server(cloud_service, server)

    SERVICE_NAMES = (
        fog_pb2.DESCRIPTOR.services_by_name["CloudService"].full_name,
        reflection.SERVICE_NAME,
    )

    # Enable reflection (for grpcui and grpcurl)
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port("[::]:50051")
    server.start()
    print("Server started on port 50051")

    task_processor_thread = threading.Thread(target=cloud_service.process_task_queue)
    task_processor_thread.start()
    print("Started Queue Processor on seperate Thread")

    server.wait_for_termination()



if __name__ == "__main__":
    serve()
