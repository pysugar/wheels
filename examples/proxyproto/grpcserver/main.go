package main

import (
	"log"
	"net"

	"github.com/pires/go-proxyproto"
	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("无法监听端口 50051：%v", err)
	}
	defer listener.Close()

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()

	grpcServer := grpc.NewServer()

	// 注册您的 gRPC 服务
	// pb.RegisterYourService(grpcServer, &YourService{})

	log.Println("gRPC 服务器正在监听端口 50051...")
	if err := grpcServer.Serve(proxyListener); err != nil {
		log.Fatalf("gRPC 服务器错误：%v", err)
	}
}
