# 第一阶段（编译）
FROM golang:1.27-alpine AS builder

WORKDIR /app

# go.mod 相当于指明了依赖的环境
# go.sum 相当于加了防伪码（哈希值）
COPY go.mod go.sum ./

RUN go mod download

COPY . .

# CGO_ENABLED=0 代表禁用 CGO，让编译文件不依赖于 C库，放到哪里都能跑
# GOOS=linux 代表 目标的 OS 是 Linux，不然这里默认的是 Windows
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-probe ./cmd/checker

# 第二阶段（运行）
FROM alpine:3.20

# tzdata 让你的程序知道“现在几点”是正确的时区；
# ca-certificates 让你的程序能相信“HTTPS 网站”是可信的。
RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

# 从第一阶段中把二进制文件拷过来
COPY --from=builder /app/ai-probe .

EXPOSE 8080

CMD ["./ai-probe"]