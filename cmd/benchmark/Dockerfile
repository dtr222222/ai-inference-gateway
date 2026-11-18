FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY . .
# 编译压测脚本
RUN go build -o benchmark main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/benchmark .
# 运行压测脚本
CMD ["./benchmark"]