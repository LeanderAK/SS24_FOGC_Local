from collections import deque
from json import dump, load
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
    def __init__(self,value, task_id=None, result= None, processed= False):
        if task_id is None:
            task_id=uuid4()
        self.id = task_id
        self.value = value
        self.result = result
        self.processed = processed

    def to_dict(self):
        return {
            "id": str(self.id),
            "value": self.value,
            "result": self.result,
            "processed": self.processed
        }

    @classmethod
    def from_dict(cls, data):
        return cls(value=data["value"], task_id=data["id"], result=data["result"], processed=data["processed"])

    def __str__(self):
        return f'Task_id: {self.id}, processed: {self.processed}'
    
class TaskQueue:
    def __init__(self):
        self.tasks = deque()
        self.lock = threading.Lock()
        self.persistence_file_path = os.path.join(os.path.dirname(__file__), "queue_replica.json")

        self.load_tasks_from_replica_queue()

    def init_persistence_file(self):
        if not os.path.exists(self.persistence_file_path):
            with open(self.persistence_file_path, "w") as f:
                print("creating replica file")
                res = dump([], f)

    def load_tasks_from_replica_queue(self):
        if os.path.exists(self.persistence_file_path):
            with open(self.persistence_file_path, "r") as f:
                backup_data = load(f)
                for task in backup_data:
                    instance_task = Task.from_dict(task)
                    self.add_task(task=instance_task)
        else:
            print("replica file not found!")

    def write_tasks_to_replica_queue(self):
        if os.path.exists(self.persistence_file_path):
            with open(self.persistence_file_path, "w") as f:
                task_list = [task.to_dict() for task in self.tasks]
                res = dump(task_list, f)
        else:
            print("replica file not found!")
                
    def add_task(self, task: Task):
        with self.lock:
            self.tasks.append(task)
            self.write_tasks_to_replica_queue()

    def add_task_to_front(self, task: Task):
        with self.lock:
            self.tasks.appendleft(task)
            self.write_tasks_to_replica_queue()

    def get_task(self) -> Task:
        with self.lock:
            if self.tasks:
                task = self.tasks.popleft()
                self.write_tasks_to_replica_queue()
                return task
            else:
                return None

    def peek_task(self) -> Task:
        with self.lock:
            if self.tasks:
                return self.tasks[0]
            else:
                return None

            
class CloudService(fog_pb2_grpc.CloudServiceServicer):
    def __init__(self):
        self.task_queue = TaskQueue()
        self.feedback_url = "placeholder_url"
        self.task_queue.init_persistence_file()

    def ProcessData(self, request: fog_pb2.ProcessDataRequest, context):
        response_data = fog_pb2.Position(x=10.0, y=20.0, z=0.5)
        print(f"Received data: {request.data}")
        if not request.ListFields():
            print("Data empty!")

        task = Task(value=request.data.value)
        self.task_queue.add_task(task=task)
        
        return fog_pb2.ProcessDataResponse()

    def process_task_queue(self):
        while True:
            task_peek = self.task_queue.peek_task()
            if task_peek:
                if task_peek.processed == False:
                    self.process_task(task=task_peek)
                response = self.send_feedback(task=task_peek)

                #TODO if result is success, remove the peeked task from queue
                self.task_queue.get_task()
                time.sleep(1)
            else:
                time.sleep(1)

    def process_task(self, task: Task):
        task_value = task.value
        
        task.result = task_value
        task.processed = True
        return task
    
    def send_feedback(self, task):
        print("sending feedback")
        channel = grpc.insecure_channel('localhost:50052')
        stub = fog_pb2_grpc.EdgeServiceStub(channel)
        
        # TODO send actual data
        result_data = {
            'x' : 4,
            'y' : 5,
            'z' : 6
        }

        request = fog_pb2.UpdatePositionRequest(position=result_data)
        try:
            response = stub.UpdatePosition(request)
            return response
        except grpc.RpcError as e:
            print("Request to Edge Service failed!")


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
