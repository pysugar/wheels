package signal_test

import (
	. "github.com/pysugar/wheels/signal"
	"testing"
)

func TestNotifierSignal(t *testing.T) {
	n := NewNotifier()

	w := n.Wait()
	n.Signal()

	select {
	case <-w:
		t.Logf("notifier signaled")
	default:
		t.Fail()
	}
}
