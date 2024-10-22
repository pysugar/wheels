package protobuf_test

import (
	"github.com/pysugar/wheels/binproto/protobuf"
	"os"
	"testing"
)

func TestParseProtoMessage(t *testing.T) {
	data := []byte{8, 1, 18, 3, 'f', 'o', 'o'}

	protobuf.ParseProtoMessage(data)
}

func TestParseAllTypes(t *testing.T) {
	data, err := os.ReadFile("/tmp/alltypes.bin")
	if err != nil {
		t.Fatalf("Failed to read file: %v\n", err)
	}

	protobuf.ParseProtoMessage(data)
}
