package ldm

import (
	"encoding/binary"
	"io"
	"net"
)

func Uint32ToBytes(n uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n)
	return buf
}

func BytesToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func SendMessage(conn net.Conn, message string) error {
	data := []byte(message)
	length := uint32(len(data))
	// 构造消息头
	header := Uint32ToBytes(length)
	// 发送消息头
	if _, err := conn.Write(header); err != nil {
		return err
	}
	// 发送消息体
	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

func ReceiveMessage(conn net.Conn) (string, error) {
	// 读取消息头（4 字节）
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	length := BytesToUint32(header)
	// 读取消息体
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return "", err
	}
	return string(data), nil
}
