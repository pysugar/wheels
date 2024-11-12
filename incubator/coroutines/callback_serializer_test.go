package coroutines_test

import (
	"context"
	"github.com/pysugar/wheels/incubator/coroutines"
	"sync"
	"testing"
)

func TestCallbackSerializer_Schedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	concurrency := 100
	serializer := coroutines.NewCallbackSerializer(ctx, concurrency)

	doneChs := make([]chan int, 0)
	num := 0
	for i := 0; i < concurrency; i++ {
		doneCh := make(chan int)
		doneChs = append(doneChs, doneCh)
		serializer.Schedule(func() {
			num += 1
			// time.Sleep(100 * time.Millisecond)
			doneCh <- num
		})
	}

	var wg sync.WaitGroup
	for i, doneCh := range doneChs {
		wg.Add(1)
		go func(idx int, numCh <-chan int) {
			defer wg.Done()
			n := <-numCh
			t.Logf("%d: %d", idx, n)
			if idx != n-1 {
				t.Fail()
			}
		}(i, doneCh)
	}
	wg.Wait()
}
