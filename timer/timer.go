package timer

import (
	"context"
	"github.com/pysugar/wheels/task"
	"sync"
	"time"
)

type ActivityUpdater interface {
	Update()
}

type ActivityTimer struct {
	sync.RWMutex
	updated   chan struct{}
	checkTask *task.Periodic
	onTimeout func()
}

func (t *ActivityTimer) Update() {
	select {
	case t.updated <- struct{}{}:
	default:
	}
}

func (t *ActivityTimer) check() error {
	select {
	case <-t.updated:
	default:
		t.finish()
	}
	return nil
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	if t.onTimeout != nil {
		t.onTimeout()
		t.onTimeout = nil
	}
	if t.checkTask != nil {
		t.checkTask.Close()
		t.checkTask = nil
	}
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	if timeout == 0 {
		t.finish()
		return
	}

	checkTask := &task.Periodic{
		Interval: timeout,
		Execute:  t.check,
	}

	t.Lock()
	if t.checkTask != nil {
		t.checkTask.Close()
	}
	t.checkTask = checkTask
	t.Unlock()

	t.Update()

	err := checkTask.Start()
	if err != nil {
		panic(err)
	}
}

func CancelAfterInactivity(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
	}
	timer.SetTimeout(timeout)
	return timer
}