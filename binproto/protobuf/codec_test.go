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

func TestParseRawProto(t *testing.T) {
	rawDesc := []byte{
		0x0a, 0x0f, 0x75, 0x73, 0x65, 0x72, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
		0x6f, 0x12, 0x1c, 0x70, 0x79, 0x73, 0x75, 0x67, 0x61, 0x72, 0x2e, 0x77, 0x68, 0x65, 0x65, 0x6c,
		0x73, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x1a,
		0x1a, 0x73, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x64, 0x5f, 0x6d, 0x65,
		0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x71, 0x0a, 0x04, 0x55,
		0x73, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01,
		0x28, 0x0d, 0x52, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61,
		0x69, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12,
		0x3d, 0x0a, 0x07, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
		0x32, 0x23, 0x2e, 0x70, 0x79, 0x73, 0x75, 0x67, 0x61, 0x72, 0x2e, 0x77, 0x68, 0x65, 0x65, 0x6c,
		0x73, 0x2e, 0x73, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x64, 0x4d, 0x65,
		0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x07, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x41,
		0x0a, 0x07, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65,
		0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65,
		0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
		0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
		0x64, 0x42, 0x73, 0x0a, 0x27, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
		0x70, 0x79, 0x73, 0x75, 0x67, 0x65, 0x72, 0x2e, 0x77, 0x68, 0x65, 0x65, 0x6c, 0x73, 0x2e, 0x65,
		0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x50, 0x01, 0x5a, 0x27,
		0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x79, 0x73, 0x75, 0x67,
		0x61, 0x72, 0x2f, 0x77, 0x68, 0x65, 0x65, 0x6c, 0x73, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
		0x65, 0x73, 0x2f, 0x75, 0x73, 0x65, 0x72, 0xaa, 0x02, 0x1c, 0x50, 0x79, 0x53, 0x75, 0x67, 0x61,
		0x72, 0x2e, 0x57, 0x68, 0x65, 0x65, 0x6c, 0x73, 0x2e, 0x45, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65,
		0x73, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
	}
	descriptor, err := protobuf.ParseRawProto(rawDesc)
	t.Log(descriptor, err)
}