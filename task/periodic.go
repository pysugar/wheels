package task

import (
	"log"
	"sync"
	"time"
)

// Periodic is a task that runs periodically.
type Periodic struct {
	// Interval of the task being run
	Interval time.Duration
	// Execute is the task function
	Execute func() error

	m       sync.Mutex
	timer   *time.Timer
	running bool
}

func (t *Periodic) hasClosed() bool {
	t.m.Lock()
	defer t.m.Unlock()

	return !t.running
}

func (t *Periodic) checkedExecute() error {
	if t.hasClosed() {
		return nil
	}

	if err := t.Execute(); err != nil {
		t.m.Lock()
		t.running = false
		t.m.Unlock()
		return err
	}

	t.m.Lock()
	defer t.m.Unlock()

	if !t.running {
		return nil
	}

	t.timer = time.AfterFunc(t.Interval, func() {
		err := t.checkedExecute()
		if err != nil {
			log.Printf("checked execute error: %v", err)
		}
	})

	return nil
}

// Start implements common.Runnable.
func (t *Periodic) Start() error {
	t.m.Lock()
	if t.running {
		t.m.Unlock()
		return nil
	}
	t.running = true
	t.m.Unlock()

	if err := t.checkedExecute(); err != nil {
		t.m.Lock()
		t.running = false
		t.m.Unlock()
		return err
	}

	return nil
}

// Close implements common.Closable.
func (t *Periodic) Close() error {
	t.m.Lock()
	defer t.m.Unlock()

	t.running = false
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}

	return nil
}
