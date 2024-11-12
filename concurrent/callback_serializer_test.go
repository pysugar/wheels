package concurrent_test

import (
	"context"
	"github.com/pysugar/wheels/concurrent"
	"sync"
	"testing"
)

func TestCallbackSerializer_Schedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serializer := concurrent.NewCallbackSerializer(ctx)

	doneChs := make([]chan int, 0)
	num := 0
	for i := 0; i < 10; i++ {
		doneCh := make(chan int)
		doneChs = append(doneChs, doneCh)
		serializer.TrySchedule(func(ctx context.Context) {
			num += 1
			// time.Sleep(500 * time.Millisecond)
			doneCh <- num
		})
	}

	var wg sync.WaitGroup
	for i, doneCh := range doneChs {
		wg.Add(1)
		go func(index int, numCh <-chan int) {
			defer wg.Done()
			n := <-numCh
			t.Logf("%d: %d", index, n)
		}(i, doneCh)
	}
	wg.Wait()
}
