package pipe

import (
	"github.com/pysugar/wheels/buf"
	"github.com/pysugar/wheels/signal"
	"github.com/pysugar/wheels/signal/done"
)

type state byte

const (
	opened state = iota
	closed
	errord
)

type pipeOption struct {
	limit           int32 // maximum buffer size in bytes
	discardOverflow bool
	onTransmission  func(buffer buf.MultiBuffer) buf.MultiBuffer
}

func (o *pipeOption) isFull(curSize int32) bool {
	return o.limit >= 0 && curSize > o.limit
}

// Option for creating new Pipes.
type Option func(*pipeOption)

// WithoutSizeLimit returns an Option for Pipe to have no size limit.
func WithoutSizeLimit() Option {
	return func(opt *pipeOption) {
		opt.limit = -1
	}
}

// WithSizeLimit returns an Option for Pipe to have the given size limit.
func WithSizeLimit(limit int32) Option {
	return func(opt *pipeOption) {
		opt.limit = limit
	}
}

func OnTransmission(hook func(mb buf.MultiBuffer) buf.MultiBuffer) Option {
	return func(option *pipeOption) {
		option.onTransmission = hook
	}
}

// DiscardOverflow returns an Option for Pipe to discard writes if full.
func DiscardOverflow() Option {
	return func(opt *pipeOption) {
		opt.discardOverflow = true
	}
}

// OptionsFromContext returns a list of Options from context.
func OptionsFromContext(limit int32) []Option {
	var opt []Option

	if limit >= 0 {
		opt = append(opt, WithSizeLimit(limit))
	} else {
		opt = append(opt, WithoutSizeLimit())
	}

	return opt
}

// New creates a new Reader and Writer that connects to each other.
func New(opts ...Option) (*Reader, *Writer) {
	p := &pipe{
		readSignal:  signal.NewNotifier(),
		writeSignal: signal.NewNotifier(),
		done:        done.New(),
		errChan:     make(chan error, 1),
		option: pipeOption{
			limit: -1,
		},
	}

	for _, opt := range opts {
		opt(&(p.option))
	}

	return &Reader{
			pipe: p,
		}, &Writer{
			pipe: p,
		}
}
