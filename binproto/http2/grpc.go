package http2

import (
	"bytes"
	"encoding/binary"
	"google.golang.org/protobuf/proto"
	"log"
)

func BuildGrpcFrame(message proto.Message) ([]byte, error) {
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
