import threading

import grpc
from concurrent import futures

from pb import chat_pb2, chat_pb2_grpc
from transformers import AutoTokenizer, AutoModelForCausalLM, TextIteratorStreamer, StoppingCriteria
from threading import Thread
import socket

path = "qwen-model"
tokenizer = AutoTokenizer.from_pretrained(path)
model = AutoModelForCausalLM.from_pretrained(path)

# 从 go 端定义的参数在哪里获取: 从 request 里获取

class CancelCriteria(StoppingCriteria):
    def __init__(self, cancel_event):
        self.cancel_event = cancel_event

    def __call__(self, input_ids, scores, **kwargs):
        return self.cancel_event.is_set()

class ChatService(chat_pb2_grpc.ChatServiceServicer):
    def StreamChat(self, request, context):
        # 测试打印信息
        print(socket.gethostname(), request.request_id)
        # 构建标准的对话信息
        messages = [
            {"role": "user", "content": request.prompt}
        ]
        # 让 tokenizer 自动套用 Qwen 的对话模板
        text = tokenizer.apply_chat_template(
            messages,
            tokenize=False,
            add_generation_prompt=True

        )
        inputs = tokenizer(text, return_tensors="pt")
        streamer = TextIteratorStreamer(tokenizer, skip_prompt=True, skip_special_tokens=True)
        cancel_event = threading.Event()
        criteria = CancelCriteria(cancel_event)
        generation_kwargs = dict(
            **inputs,
            streamer=streamer,
            max_new_tokens=request.max_new_tokens if request.max_new_tokens > 0 else 20,
            stopping_criteria=[criteria]
        )
        # 这个 thread 会再开一个线程跑模型
        thread = Thread(target=model.generate, kwargs=generation_kwargs)
        thread.start()
        while True:
            try:
                token = next(streamer)
                if not context.is_active():
                    cancel_event.set()
                    break
                yield chat_pb2.ChatResponse(
                    request_id=request.request_id,
                    token=token,
                    is_finish=False,
                    error_msg=""
                )
            except StopIteration:
                yield chat_pb2.ChatResponse(
                    request_id=request.request_id,
                    token="",
                    is_finish=True,
                    error_msg=""
                )
                break

def serve():
    # grpc.server 用来创建 gRPC服务器实例， 线性池：futures.ThreadPoolExecutor(max_workers=10)，总共准备了10个线程
    # 比如 go 端发来十个并发请求，grpc 会分配这 10 个线程去分别执行stream chat函数
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    # 这个函数的作用是绑定，以后收到的请求都交给类里的 StreamChat方法处理
    chat_pb2_grpc.add_ChatServiceServicer_to_server(ChatService(), server)
    # 绑定端口，这里的 insecure表示不加密（明文传输），但之后必须得改
    server.add_insecure_port('[::]:50051')
    # 开始监听窗口
    server.start()
    # 如果没有这一句，程序执行完 start() 后就会直接退出（进程结束）。加上这句，主线程就会在这里死等（挂起），直到你按 Ctrl+C 把服务器关掉。
    server.wait_for_termination()

if __name__ == '__main__':
    serve()