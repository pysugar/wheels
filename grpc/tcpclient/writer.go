package tcpclient

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/net/http2/hpack"
	"io"
	"log"
)

func (c *grpcClient) writeHeadersFrame(streamID uint32, headers []hpack.HeaderField) error {
	blockFragment := c.encodeHpackHeaders(headers)
	return WriteHeadersFrame(c.conn, streamID, FlagHeadersEndHeaders, blockFragment)
}

func WriteDataFrame(w io.Writer, streamID uint32, flags Flags, body []byte) error {
	length := len(body)
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x0                                  // Type: DATA (0x0)
	buf[4] = uint8(flags)                         // Flags
	binary.BigEndian.PutUint32(buf[5:], streamID) // Stream ID: 1

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write data header success, length = %d\n", n)
	}
	if length > 0 {
		if n, err := w.Write(body); err != nil {
			return err
		} else {
			log.Printf("Write data payload success, length = %d\n", n)
		}
	}
	return nil
}

func WriteHeadersFrame(w io.Writer, streamID uint32, flags Flags, blockFragment []byte) error {
	length := len(blockFragment)
	var buf [http2frameHeaderLen]byte
	// binary.BigEndian.PutUint32(buf[:4], uint32(length))
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x1                                  // Type: HEADERS (0x1)
	buf[4] = uint8(flags)                         // Flags: END_HEADERS
	binary.BigEndian.PutUint32(buf[5:], streamID) // Stream ID (client-initiated, must be odd)

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write headers header success, length = %d\n", n)
	}
	if n, err := w.Write(blockFragment); err != nil {
		return err
	} else {
		log.Printf("Write headers payload success, length = %d\n", n)
	}
	return nil
}

func WriteSettingsFrame(w io.Writer, flags Flags, payload []byte) error {
	length := len(payload)
	// Frame Header: Length (3 bytes), Type (1 byte), Flags (1 byte), Stream Identifier (4 bytes)
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x4                             // Type: SETTINGS (0x4)
	buf[4] = uint8(flags)                    // Flags
	binary.BigEndian.PutUint32(buf[5:], 0x0) // Stream ID

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write settings header success, length = %d\n", n)
	}
	if len(payload) > 0 {
		if n, err := w.Write(payload); err != nil {
			return err
		} else {
			log.Printf("Write settings payload success, length = %d\n", n)
		}
	}
	return nil
}

func WritePingFrame(w io.Writer, flags Flags, payload []byte) error {
	length := len(payload)
	if length != 8 {
		return fmt.Errorf("ping payload should be 8")
	}
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x6                             // Type: PING (0x6)
	buf[4] = uint8(flags)                    // Flags
	binary.BigEndian.PutUint32(buf[5:], 0x0) // Stream ID

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write ping header success, length = %d\n", n)
	}
	if n, err := w.Write(payload[:]); err != nil {
		return err
	} else {
		log.Printf("Write ping payload success, length = %d\n", n)
	}

	return nil
}
