package protobuf

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func ParseRawProto(rawDesc []byte) (*descriptorpb.FileDescriptorProto, error) { // (protoreflect.FileDescriptor, error) {
	var err error
	decompressedBytes := rawDesc
	if isGzipCompressed(rawDesc) {
		r, er := gzip.NewReader(bytes.NewReader(rawDesc))
		if er != nil {
			log.Printf("failed to create gzip reader: %v", er)
			return nil, er
		}
		defer r.Close()

		decompressedBytes, err = io.ReadAll(r)
		if err != nil {
			log.Printf("failed to decompress proto_rawDesc: %v", err)
			return nil, err
		}
		log.Printf("decompress data success (%d -> %d)\n", len(rawDesc), len(decompressedBytes))
	}

	fileDescriptor := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(decompressedBytes, fileDescriptor)
	if err != nil {
		log.Printf("failed to unmarshal proto_rawDesc: %v", err)
		return nil, err
	}

	descriptorJSON, err := json.MarshalIndent(fileDescriptor, "", "  ")
	if err != nil {
		log.Printf("failed to marshal descriptor to JSON: %v", err)
	}
	fmt.Printf("File Descriptor JSON:\n%s\n", descriptorJSON)

	fmt.Printf("\nRecovered .proto file:\n")
	fmt.Printf("syntax = \"%s\";\n", fileDescriptor.GetSyntax())
	fmt.Printf("package %s;\n\n", fileDescriptor.GetPackage())

	for _, messageType := range fileDescriptor.GetMessageType() {
		fmt.Printf("message %s {\n", messageType.GetName())
		for _, field := range messageType.GetField() {
			fmt.Printf("  %s %s = %d;\n", field.GetType().String(), field.GetName(), field.GetNumber())
		}
		fmt.Printf("}\n")
	}

	return fileDescriptor, nil
}

func isGzipCompressed(data []byte) bool {
	return len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b
}
