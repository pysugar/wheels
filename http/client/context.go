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
	WebSocket
)

var (
	protocolCtxKey = &contextKey{"protocol"}
	verboseCtxKey  = &contextKey{"verbose"}
	upgradeCtxKey  = &contextKey{"upgrade"}
	gorillaCtxKey  = &contextKey{"gorilla"}
	insecureCtxKey = &contextKey{"insecure"}
)

func (hp HttpProtocol) String() string {
	switch hp {
	case HTTP2:
		return "HTTP2"
	case HTTP1:
		return "HTTP1"
	case HTTP10:
		return "HTTP10"
	case HTTP11:
		return "HTTP11"
	case WebSocket:
		return "WebSocket"
	default:
		return "Unknown"
	}
}

func WithProtocol(ctx context.Context, protocol HttpProtocol) context.Context {
	return context.WithValue(ctx, protocolCtxKey, protocol)
}

func WithVerbose(ctx context.Context) context.Context {
	return context.WithValue(ctx, verboseCtxKey, true)
}

func WithUpgrade(ctx context.Context) context.Context {
	return context.WithValue(ctx, upgradeCtxKey, true)
}

func WithGorilla(ctx context.Context) context.Context {
	return context.WithValue(ctx, gorillaCtxKey, true)
}

func WithInsecure(ctx context.Context) context.Context {
	return context.WithValue(ctx, insecureCtxKey, true)
}

func ProtocolFromContext(ctx context.Context) HttpProtocol {
	if protocol, ok := ctx.Value(protocolCtxKey).(HttpProtocol); ok {
		return protocol
	}
	return Unknown
}

func VerboseFromContext(ctx context.Context) bool {
	_, ok := ctx.Value(verboseCtxKey).(bool)
	return ok
}

func UpgradeFromContext(ctx context.Context) bool {
	_, ok := ctx.Value(upgradeCtxKey).(bool)
	return ok
}

func GorillaFromContext(ctx context.Context) bool {
	_, ok := ctx.Value(gorillaCtxKey).(bool)
	return ok
}

func InsecureFromContext(ctx context.Context) bool {
	_, ok := ctx.Value(insecureCtxKey).(bool)
	return ok
}
