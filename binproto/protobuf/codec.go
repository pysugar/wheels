package protobuf

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

func readVarint(r io.Reader) (int, error) {
	var result int
	var shift uint
	for {
		var b [1]byte
		_, err := r.Read(b[:])
		if err != nil {
			return 0, err
		}

		result |= int(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint too long")
		}
	}
	return result, nil
}

func readFieldHeader(r io.Reader) (int, int, error) {
	tag, err := readVarint(r)
	if err != nil {
		return 0, 0, err
	}
	fieldNumber := tag >> 3
	wireType := tag & 0x7
	return fieldNumber, wireType, nil
}

func readFull(r io.Reader, buf []byte) error {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		if err != nil {
			return err
		}
		total += n
	}
	return nil
}

func readValue(r io.Reader, wireType int) (interface{}, error) {
	switch wireType {
	case 0: // Varint
		return readVarint(r)
	case 1: // 64-bit
		var b [8]byte
		err := readFull(r, b[:])
		if err != nil {
			return nil, err
		}
		return int64(binary.LittleEndian.Uint64(b[:])), nil
	case 2: // Length-delimited
		length, err := readVarint(r)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, length)
		err = readFull(r, buf)
		if err != nil {
			return nil, err
		}
		return string(buf), nil
	case 5: // 32-bit
		var b [4]byte
		err := readFull(r, b[:])
		if err != nil {
			return nil, err
		}
		return int32(binary.LittleEndian.Uint32(b[:])), nil
	default:
		return nil, fmt.Errorf("unsupported wire type: %d", wireType)
	}
}

func ParseProtoMessage(data []byte) {
	r := strings.NewReader(string(data))
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
