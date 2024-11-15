package client

import "time"

// dialOptions configure a Dial call. dialOptions are set by the DialOption values passed to Dial.
type dialOptions struct {
	// authority string
	useTLS  bool
	timeout time.Duration
}

type DialOption func(*dialOptions)

var (
	defaultDialOptions = &dialOptions{
		timeout: 30 * time.Second,
	}
)

func WithTLS() DialOption {
	return func(o *dialOptions) {
		o.useTLS = true
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
