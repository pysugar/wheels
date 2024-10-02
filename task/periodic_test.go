package task_test

import (
	. "github.com/pysugar/wheels/task"
	"testing"
	"time"
)

func TestPeriodicTaskStop(t *testing.T) {
	value := 0
	task := &Periodic{
		Interval: time.Second * 2,
		Execute: func() error {
			value++
			t.Logf("periodic triggered, value: %d", value)
			return nil
		},
	}
	err := task.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 5)

	err = task.Close()
	if err != nil {
		t.Fatal(err)
	}
	if value != 3 {
		t.Fatal("expected 3, but got ", value)
	}
	time.Sleep(time.Second * 4)
	if value != 3 {
		t.Fatal("expected 3, but got ", value)
	}

	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 3)
	if value != 5 {
		t.Fatal("Expected 5, but ", value)
	}

	err = task.Close()
	if err != nil {
		t.Fatal(err)
	}
}
