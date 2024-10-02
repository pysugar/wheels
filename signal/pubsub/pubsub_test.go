package pubsub_test

import (
	. "github.com/pysugar/wheels/signal/pubsub"
	"testing"
)

func TestPubsub(t *testing.T) {
	service := NewService()

	sub := service.Subscribe("a")
	service.Publish("a", 1)

	select {
	case v := <-sub.Wait():
		if v != 1 {
			t.Error("expected subscribed value 1, but got ", v)
		}
	default:
		t.Fail()
	}

	sub.Close()
	service.Publish("a", 2)

	select {
	case <-sub.Wait():
		t.Fail()
	default:
	}

	service.Cleanup()
}
