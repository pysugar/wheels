package coroutines

import "context"

type CallbackSerializer struct {
	ch chan func()
}

// NewCallbackSerializer creates a new Serializer
func NewCallbackSerializer(ctx context.Context, channelSize int) *CallbackSerializer {
	cs := &CallbackSerializer{
		ch: make(chan func(), channelSize),
	}
	go cs.loop(ctx)
	return cs
}

// Schedule adds a function to the serializer queue
func (cs *CallbackSerializer) Schedule(f func()) {
	cs.ch <- f
}

// loop runs functions from the queue in order
func (cs *CallbackSerializer) loop(ctx context.Context) {
	for {
		select {
		case f := <-cs.ch:
			if f != nil {
				f()
			}
		case <-ctx.Done():
			return
		}
	}
}
