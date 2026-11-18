package main

import (
	chat "ai-inference-gateway/pb"
	"context"
	"fmt"
	"io"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// 注意：这里的 pb 替换成你生成 go 文件的包路径
	// 假设你在项目里建了一个 pb 文件夹，生成的 chat.pb.go 都在里面
	// pb "你的项目/pb"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	client := chat.NewChatServiceClient(conn)

	req := &chat.ChatRequest{
		RequestId:    "test001",
		Prompt:       "写一首关于思念的诗",
		MaxNewTokens: 200,
		Temperature:  1,
	}

	stream, err := client.StreamChat(context.Background(), req)
	if err != nil {
		log.Fatalf("调用失败: %v", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("\n[流结束]")
			break
		}
		if err != nil {
			log.Fatalf("接收失败: %v", err)
		}

		// 处理 gRPC 返回的错误
		if res.ErrorMsg != "" {
			log.Printf("服务端报错: %s", res.ErrorMsg)
			break
		}

		// 打印 token
		fmt.Print(res.Token)
	}
}
