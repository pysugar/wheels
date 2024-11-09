package extensions

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggingUnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Printf("[%s] Incoming Metadata: %v\n", info.FullMethod, md)
	}

	log.Printf("< [%s] Received RPC Request: %+v\n", info.FullMethod, req)
	resp, err := handler(ctx, req)
	log.Printf("[%s] Sending RPC Response: %+v, Err: %v >\n", info.FullMethod, resp, err)
	return resp, err
}
