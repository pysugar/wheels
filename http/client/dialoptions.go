package client

import (
	"net"
	"time"
)

// dialOptions configure a Dial call. dialOptions are set by the DialOption values passed to Dial.
type dialOptions struct {
	// authority string
	useTLS  bool
	timeout time.Duration
	verbose bool
	conn    net.Conn
}

type DialOption func(*dialOptions)

var (
	defaultDialOptions = &dialOptions{
		timeout: 30 * time.Second,
		verbose: false,
	}
)

func WithTLS() DialOption {
	return func(o *dialOptions) {
		o.useTLS = true
	}
}

func WithTimeout(timeout time.Duration) DialOption {
	return func(o *dialOptions) {
		o.timeout = timeout
	}
}

func WithConn(conn net.Conn) DialOption {
	return func(o *dialOptions) {
		o.conn = conn
	}
}

func WithVerbose() DialOption {
	return func(o *dialOptions) {
		o.verbose = true
	}
}

func evaluateOptions(opts []DialOption) *dialOptions {
	optCopy := &dialOptions{}
	*optCopy = *defaultDialOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}
