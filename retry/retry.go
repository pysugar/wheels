package retry

import (
	"github.com/pysugar/wheels/errors"
	"time"
)

var ErrRetryFailed = errors.New("retry failed")

// Strategy is a way to retry on a specific function.
type Strategy interface {
	// On performs a retry on a specific function, until it doesn't return any error.
	On(func() error) error
}

type retryer struct {
	totalAttempt int
	nextDelay    func() uint32
}

// On implements Strategy.On.
func (r *retryer) On(method func() error) error {
	attempt := 0
	accumulatedErrors := make([]error, 0, r.totalAttempt)
	for attempt < r.totalAttempt {
		err := method()
		if err == nil {
			return nil
		}
		numErrors := len(accumulatedErrors)
		if numErrors == 0 || err.Error() != accumulatedErrors[numErrors-1].Error() {
			accumulatedErrors = append(accumulatedErrors, err)
		}
		delay := r.nextDelay()
		time.Sleep(time.Duration(delay) * time.Millisecond)
		attempt++
	}
	return errors.Multi(ErrRetryFailed, accumulatedErrors)
}

// Timed returns a retry strategy with fixed interval.
func Timed(attempts int, delay uint32) Strategy {
	return &retryer{
		totalAttempt: attempts,
		nextDelay: func() uint32 {
			return delay
		},
	}
}

func ExponentialBackoff(attempts int, delay uint32) Strategy {
	nextDelay := uint32(0)
	return &retryer{
		totalAttempt: attempts,
		nextDelay: func() uint32 {
			r := nextDelay
			nextDelay += delay
			return r
		},
	}
}
