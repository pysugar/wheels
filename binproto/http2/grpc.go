package http2

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"google.golang.org/protobuf/proto"
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

	messageData := data[5 : 5+messageLength]

	if compressedFlag != 0 {
		decompressedData, err := decompress(messageData, "gzip")
		if err != nil {
			return fmt.Errorf("failed to decompress grpc message: %w", err)
		}
		messageData = decompressedData
		return fmt.Errorf("compressed responses are not supported in this client")
	}

	err := proto.Unmarshal(messageData, message)
	if err != nil {
		log.Printf("Failed to unmarshal response: %v", err)
		return err
	}
	return nil
}

func compress(data []byte, algo string) ([]byte, error) {
	switch algo {
	case "gzip":
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		_, err := writer.Write(data)
		if err != nil {
			return nil, fmt.Errorf("failed to write gzip data: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}
}

func decompress(data []byte, algo string) ([]byte, error) {
	switch algo {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		decompressedData, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read gzip data: %w", err)
		}
		return decompressedData, nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}
}
