from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline

app = FastAPI()

# 跑压测用 sshleifer/tiny-gpt2
generator = pipeline("text-generation", model="/app/qwen-model")


class GenRequest(BaseModel):
    prompts: list[str]
    request_ids: list[str]


@app.post("/prompt")
def generate_text(request: GenRequest):
    # 调用模型生成文本，传的是一个列表，用于动态批处理
    limit = min(len(request.prompts), 5)
    texts = []
    for i in range(limit):
        messages = [
            {"role": "user", "content": request.prompts[i]},
        ]
        reply = generator(messages, max_new_tokens=20)
        texts.append(reply[0]["generated_text"][-1]['content'])
    return {"results": texts, "request_ids": request.request_ids[:limit]}

# 压测代码
# @app.post("/prompt")
# def generate_text(request: GenRequest):
#     # 调用模型生成文本，传的是一个列表，用于动态批处理
#     limit = min(len(request.prompts), 50)
#     raw_results = generator(request.prompts[:limit], max_new_tokens=50)
#     # 使用列表表达式来提取模型的回复
#     texts = [res[0]["generated_text"] for res in raw_results]
#     return {"results": texts, "request_ids": request.request_ids[:limit]}