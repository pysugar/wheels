package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func EncodeGrpcPayload(payload []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x00) // 标志位（0 表示未压缩）
	length := uint32(len(payload))
	_ = binary.Write(&buf, binary.BigEndian, length)
	buf.Write(payload)
	return buf.Bytes()
}

func DecodeGrpcPayload(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("invalid grpc frame data")
	}
	compressedFlag := data[0]
	if compressedFlag != 0 {
		return nil, fmt.Errorf("compressed responses are not supported in this client")
	}
	messageLength := binary.BigEndian.Uint32(data[1:5])
	messageData := data[5 : 5+messageLength]
	return messageData, nil
}
