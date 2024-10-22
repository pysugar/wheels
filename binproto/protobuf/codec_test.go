package protobuf_test

import (
	"github.com/pysugar/wheels/binproto/protobuf"
	"os"
	"testing"
)

func TestParseProtoMessage(t *testing.T) {
	data := []byte{8, 1, 18, 3, 'f', 'o', 'o'}
	protobuf.ParseProtoMessage(data)

	data2 := []byte{10, 12, 78, 101, 115, 116, 101, 100, 82, 97, 110, 100, 111, 109, 16, 16}
	protobuf.ParseProtoMessage(data2)
}

func TestParseAllTypes(t *testing.T) {
	data, err := os.ReadFile("/tmp/alltypes.bin")
	if err != nil {
		t.Fatalf("Failed to read file: %v\n", err)
	}
	protobuf.ParseProtoMessage(data)
}
