package task_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/pysugar/wheels/task"
)

func TestExecuteParallel(t *testing.T) {
	err := Run(context.Background(),
		func() error {
			time.Sleep(time.Millisecond * 200)
			t.Log("task1 return test error")
			return errors.New("test")
		}, func() error {
			time.Sleep(time.Millisecond * 500)
			t.Log("task2 return test2 error")
			return errors.New("test2")
		})

	if err == nil {
		t.Fatal("error should occurred")
	}
	if r := cmp.Diff(err.Error(), "test"); r != "" {
		t.Error(r)
	}
}

func TestExecuteParallelContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := Run(ctx, func() error {
		time.Sleep(time.Millisecond * 2000)
		t.Log("task1 return test error")
		return errors.New("test")
	}, func() error {
		time.Sleep(time.Millisecond * 5000)
		t.Log("task2 return test2 error")
		return errors.New("test2")
	}, func() error {
		cancel()
		t.Log("task3 trigger context cancel")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatal("error should occurred")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "canceled") {
		t.Error("expected error string to contain 'canceled', but actually not: ", errStr)
	}
}

func BenchmarkExecuteOne(b *testing.B) {
	noop := func() error {
		return nil
	}
	for i := 0; i < b.N; i++ {
		err := Run(context.Background(), noop)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteTwo(b *testing.B) {
	noop := func() error {
		return nil
	}
	for i := 0; i < b.N; i++ {
		err := Run(context.Background(), noop, noop)
		if err != nil {
			b.Fatal(err)
		}
	}
}
