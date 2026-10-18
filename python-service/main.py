from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline

app = FastAPI()
generator = pipeline("text-generation", model="sshleifer/tiny-gpt2")


class GenRequest(BaseModel):
    prompt: str


@app.post("/prompt")
def generate_text(request: GenRequest):
    # 调用模型生成文本
    result = generator(request.prompt, max_length=50)
    return {"result": result}
