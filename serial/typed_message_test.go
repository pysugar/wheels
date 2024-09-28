package serial_test

import (
	. "github.com/pysugar/wheels/serial"
	"testing"
)

func TestGetInstance(t *testing.T) {
	p, err := GetInstance("")
	if p != nil {
		t.Error("expected nil instance, but got ", p)
	}
	if err == nil {
		t.Error("expect non-nil error, but got nil")
	}
}

func TestConvertingNilMessage(t *testing.T) {
	x := Encode(nil)
	if x != nil {
		t.Error("expect nil, but actually not")
	}
}
