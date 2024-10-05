package buf_test

import (
	"crypto/tls"
	"io"
	"testing"

	. "github.com/pysugar/wheels/buf"
	"github.com/pysugar/wheels/net"
	"github.com/pysugar/wheels/testing/servers/tcp"
)

func TestWriterCreation(t *testing.T) {
	tcpServer := tcp.Server{}
	dest, err := tcpServer.Start()
	if err != nil {
		t.Fatal("failed to start tcp server: ", err)
	}
	defer tcpServer.Close()
	t.Logf("destination: %v", dest)

	conn, err := net.Dial("tcp", dest.NetAddr())
	if err != nil {
		t.Fatal("failed to dial a TCP connection: ", err)
	}
	defer conn.Close()
	t.Logf("connection: (%v -> %v)", conn.LocalAddr(), conn.RemoteAddr())

	{
		writer := NewWriter(conn)
		if _, ok := writer.(*BufferToBytesWriter); !ok {
			t.Fatal("writer is not a BufferToBytesWriter")
		}

		writer2 := NewWriter(writer.(io.Writer))
		if writer2 != writer {
			t.Fatal("writer is not reused")
		}
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})
	defer tlsConn.Close()
	t.Logf("tls connection: (%v -> %v)", tlsConn.LocalAddr(), tlsConn.RemoteAddr())
	{
		writer := NewWriter(tlsConn)
		if _, ok := writer.(*SequentialWriter); !ok {
			t.Fatal("writer is not a SequentialWriter")
		}
	}
}