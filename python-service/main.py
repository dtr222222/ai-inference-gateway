from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline

app = FastAPI()
generator = pipeline("text-generation", model="sshleifer/tiny-gpt2")


class GenRequest(BaseModel):
    prompts: list[str]


@app.post("/prompt")
def generate_text(request: GenRequest):
    # 调用模型生成文本
    raw_results = generator(request.prompts, max_new_tokens=50)
    texts = [res[0]["generated_text"] for res in raw_results]
    return {"results": texts}


