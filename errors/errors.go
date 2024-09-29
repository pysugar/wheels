package errors

import "errors"

var New = errors.New

type baseErr struct {
	base  error
	inner error
}

func (e *baseErr) Unwrap() error {
	return e.inner
}

func (e *baseErr) Error() string {
	return e.base.Error()
}

func Single(base, cause error) error {
	return &baseErr{
		base:  base,
		inner: cause,
	}
}

func Cause(err error) error {
	if err == nil {
		return nil
	}
L:
	for {
		switch inner := err.(type) {
		case interface{ Unwrap() error }:
			if inner.Unwrap() == nil {
				break L
			}
			err = inner.Unwrap()
		default:
			break L
		}
	}
	return err
}
