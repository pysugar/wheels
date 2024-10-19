package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/grpc_channelz_v1"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

const (
	address           = "localhost:50051"
	heartbeatInterval = 10 * time.Second
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	// 配置客户端的 keepalive 参数
	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(kaParams),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpcprometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpcprometheus.StreamClientInterceptor),
	)
	if err != nil {
		logger.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	channelzClient := grpc_channelz_v1.NewChannelzClient(conn)

	go startPrometheus(logger)

	// 启动心跳检测
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if er := doHeartbeat(client, logger); er != nil {
				continue
			}
			reportChannelzInfo(channelzClient)
		}
	}
}

func startPrometheus(logger grpclog.LoggerV2) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Serving metrics on :9093/metrics")
	if err := http.ListenAndServe(":9093", nil); err != nil {
		logger.Fatalf("Failed to serve metrics: %v", err)
	}
}

func doHeartbeat(client grpc_health_v1.HealthClient, logger grpclog.LoggerV2) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &grpc_health_v1.HealthCheckRequest{Service: "my_service"}
	resp, err := client.Check(ctx, req)
	if err != nil {
		logger.Errorf("Health check failed: %v", err)
		return err
	}

	logger.Infof("Health check status: %v", resp.Status)
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		logger.Warningln("Service not serving")
	}
	return nil
}

func reportChannelzInfo(client grpc_channelz_v1.ChannelzClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &grpc_channelz_v1.GetTopChannelsRequest{}
	resp, err := client.GetTopChannels(ctx, req)
	if err != nil {
		fmt.Printf("Failed to get top channels: %v\n", err)
		return
	}

	for _, ch := range resp.Channel {
		fmt.Printf("Channel ID: %d, State: %v, Target: %s\n", ch.Ref.ChannelId, ch.Data.State, ch.Data.Target)
		getSubchannelInfo(ctx, client, ch)
	}
}

func getSubchannelInfo(ctx context.Context, client grpc_channelz_v1.ChannelzClient, ch *grpc_channelz_v1.Channel) {
	for _, subChRef := range ch.SubchannelRef {
		subChReq := &grpc_channelz_v1.GetSubchannelRequest{
			SubchannelId: subChRef.SubchannelId,
		}
		subChResp, err := client.GetSubchannel(ctx, subChReq)
		if err != nil {
			fmt.Printf("Failed to get subchannel: %v\n", err)
			continue
		}
		subCh := subChResp.Subchannel
		fmt.Printf("  SubChannel ID: %d, State: %v\n", subCh.Ref.SubchannelId, subCh.Data.State)

		// 获取套接字信息
		getSocketInfo(ctx, client, subCh)
	}
}

func getSocketInfo(ctx context.Context, client grpc_channelz_v1.ChannelzClient, subCh *grpc_channelz_v1.Subchannel) {
	for _, sockRef := range subCh.SocketRef {
		sockReq := &grpc_channelz_v1.GetSocketRequest{
			SocketId: sockRef.SocketId,
		}
		sockResp, err := client.GetSocket(ctx, sockReq)
		if err != nil {
			fmt.Printf("Failed to get socket: %v\n", err)
			continue
		}
		sock := sockResp.Socket
		fmt.Printf("    Socket ID: %d, Local: %v, Remote: %v\n", sock.Ref.SocketId, sock.Local, sock.Remote)
	}
}
