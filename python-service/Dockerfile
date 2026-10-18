FROM python:3.11-slim

# 设置镜像源，hugging face 的transformers中的代码会获取这个环境变量
ENV HF_ENDPOINT=https://hf-mirror.com

# 设置不生成 .pyc 文件，让控制台输出直接打印（不缓存）
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt .

# 把 pip 下载的缓存文件删除
RUN pip install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple

COPY . .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]