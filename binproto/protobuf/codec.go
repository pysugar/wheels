package protobuf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func readVarint(r io.ByteReader) (int, error) {
	var result int
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= int(b&0x7F) << shift
		if b&0x80 == 0 { // if b < 0x80 {
			break
		}

		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint too long")
		}
	}
	return result, nil
}

func readFieldHeader(r io.ByteReader) (int, int, error) {
	tag, err := readVarint(r)
	if err != nil {
		return 0, 0, err
	}
	fieldNumber := tag >> 3
	wireType := tag & 0x7
	return fieldNumber, wireType, nil
}

func readValue(r *bytes.Reader, wireType int) (interface{}, error) {
	switch wireType {
	case 0: // Varint
		return readVarint(r)
	case 1: // 64-bit
		var value uint64
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			return nil, err
		}
		return value, nil
	case 2: // Length-delimited
		length, err := readVarint(r)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, length)
		if _, er := io.ReadFull(r, buf); er != nil {
			return nil, er
		}
		return string(buf), nil
	case 5: // 32-bit
		var value uint32
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported wire type: %d", wireType)
	}
}

func ParseProtoMessage(data []byte) {
	r := bytes.NewReader(data)
	for {
		fieldNumber, wireType, err := readFieldHeader(r)
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println("Error reading field header:", err)
			break
		}

		value, err := readValue(r, wireType)
		if err != nil {
			fmt.Println("Error reading value:", err)
			break
		}

		fmt.Printf("Field %d: %v\n", fieldNumber, value)
	}
}
