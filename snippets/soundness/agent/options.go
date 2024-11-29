package agent

import "time"

type options struct {
	heartbeatInterval time.Duration
	heartbeatPath     string
	collectPath       string
	statusFile        string
}

type Option func(*options)

var (
	defaultOptions = &options{
		heartbeatInterval: 60 * time.Second,
		heartbeatPath:     "/heartbeat",
		collectPath:       "/collect",
		statusFile:        "/tmp/status.json",
	}
)

func WithHeartbeatPath(path string) Option {
	return func(o *options) {
		o.heartbeatPath = path
	}
}

func WithCollectURL(path string) Option {
	return func(o *options) {
		o.collectPath = path
	}
}

func WithHeartbeatInterval(interval time.Duration) Option {
	return func(o *options) {
		o.heartbeatInterval = interval
	}
}

func WithStatusFile(path string) Option {
	return func(o *options) {
		o.statusFile = path
	}
}

func evaluateOptions(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}
