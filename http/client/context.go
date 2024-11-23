package client

import "context"

type (
	contextKey struct {
		name string
	}
	HttpProtocol uint8
)

const (
	Unknown HttpProtocol = iota
	HTTP2
	HTTP1
	HTTP10
	HTTP11
)

var (
	protocolCtxKey = &contextKey{"protocol"}
)

func WithProtocol(ctx context.Context, protocol HttpProtocol) context.Context {
	return context.WithValue(ctx, protocolCtxKey, protocol)
}

func ProtocolFromContext(ctx context.Context) HttpProtocol {
	if protocol, ok := ctx.Value(protocolCtxKey).(HttpProtocol); ok {
		return protocol
	}
	return Unknown
}
