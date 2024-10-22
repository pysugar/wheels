package proto

import (
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/pysugar/wheels/binproto/protobuf"
	"google.golang.org/protobuf/proto"
)

func TestParseRawDesc(t *testing.T) {
	rawDesc := file_proto_example_proto_rawDescData
	t.Logf("desc length: %d\n", len(rawDesc))
	desc, err := protobuf.ParseRawProto(rawDesc)
	t.Log(desc, err)
}

func TestMarshalExampleProto(t *testing.T) {
	example := &AllTypes{
		FieldInt32:    1,
		FieldInt64:    2,
		FieldUint32:   3,
		FieldUint64:   4,
		FieldSint32:   5,
		FieldSint64:   6,
		FieldFixed32:  7,
		FieldFixed64:  8,
		FieldSfixed32: 9,
		FieldSfixed64: 10,
		FieldFloat:    3.14,
		FieldDouble:   2.718,
		FieldBool:     true,
		FieldString:   "string_field",
		FieldBytes:    []byte{'h', 'e', 'l', 'l', 'o'},
		FieldEnum:     Status(rand.Intn(3)),
		FieldNestedMessage: &AllTypes_NestedMessage{
			NestedField: "NestedRandom",
			NestedValue: 16,
		},
		FieldRepeatedInt32:  []int32{30, 31, 32},
		FieldRepeatedString: []string{"String1", "String2", "String3"},
		FieldMap: map[string]int32{
			"Key1": 33,
			"Key2": 34,
		},
		OptionalValue: &AllTypes_OptInt32{OptInt32: 35},
	}

	data, err := proto.Marshal(example)
	if err != nil {
		panic(err)
	}

	log.Println("Data serialized and written to alltypes_data.bin")

	file, err := os.Create("/tmp/alltypes.bin")
	if err != nil {
		log.Fatalf("Failed to create file: %v\n", err)
	}
	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		log.Fatalf("Failed to write data to file: %v\n", err)
	}

	log.Printf("Proto object successfully written to port_list.bin, length: %d", n)
}
