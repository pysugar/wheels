package serial_test

import (
	. "github.com/pysugar/wheels/serial"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"testing"
)

func TestGetInstance(t *testing.T) {
	protoregistry.GlobalTypes.RangeExtensions(func(et protoreflect.ExtensionType) bool {
		t.Log(et)
		return true
	})
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		t.Log(mt.Descriptor().FullName())
		return true
	})

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
