from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline

app = FastAPI()
generator = pipeline("text-generation", model="sshleifer/tiny-gpt2")


class GenRequest(BaseModel):
    prompts: list[str]


@app.post("/prompt")
def generate_text(request: GenRequest):
    # 调用模型生成文本，传的是一个列表，用于动态批处理
    raw_results = generator(request.prompts, max_new_tokens=50)
    # 使用列表表达式来提取模型的回复
    texts = [res[0]["generated_text"] for res in raw_results]
    return {"results": texts}


