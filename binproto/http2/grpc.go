package http2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"google.golang.org/protobuf/proto"
)

func EncodeGrpcFrame(message proto.Message) ([]byte, error) {
	protoData, err := proto.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to serialize request: %v", err)
		return nil, err
	}

	length := uint32(len(protoData))
	var buf bytes.Buffer
	buf.WriteByte(0)                                   // 压缩标志（0 表示未压缩）
	err = binary.Write(&buf, binary.BigEndian, length) // 写入消息的长度
	if err != nil {
		return nil, err
	}
	buf.Write(protoData) // 写入实际的 Protobuf 消息
	return buf.Bytes(), nil
}

func DecodeGrpcFrame(data []byte, message proto.Message) error {
	if len(data) < 5 {
		return fmt.Errorf("invalid grpc frame data")
	}
	compressedFlag := data[0]
	messageLength := binary.BigEndian.Uint32(data[1:5])

	if compressedFlag != 0 {
		return fmt.Errorf("compressed responses are not supported in this client example")
	}

	messageData := data[5 : 5+messageLength]

	err := proto.Unmarshal(messageData, message)
	if err != nil {
		log.Printf("Failed to unmarshal response: %v", err)
		return err
	}
	return nil
}
