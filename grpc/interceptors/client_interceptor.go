package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggingUnaryClientInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		log.Printf("[%s] Outgoing Metadata: %v\n", method, md)
	}

	log.Printf("< [%s] Sending RPC Request: %+v\n", method, req)
	err := invoker(ctx, method, req, reply, cc, opts...)
	log.Printf("[%s] Received RPC Response: %+v, Err: %v >\n", method, reply, err)
	return err
}
