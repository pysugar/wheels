package protobuf_test

import (
	"github.com/pysugar/wheels/binproto/protobuf"
	"testing"
)

func TestParseProtoMessage(t *testing.T) {
	data := []byte{8, 1, 18, 3, 'f', 'o', 'o'}

	protobuf.ParseProtoMessage(data)
}
