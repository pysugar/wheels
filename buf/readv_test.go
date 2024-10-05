//go:build !wasm
// +build !wasm

package buf_test

import (
	"crypto/rand"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/pysugar/wheels/buf"
	"github.com/pysugar/wheels/testing/servers/tcp"
	"golang.org/x/sync/errgroup"
)

func TestReadvReader(t *testing.T) {
	tcpServer := &tcp.Server{
		MsgProcessor: func(b []byte) []byte {
			return b
		},
	}
	dest, err := tcpServer.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer tcpServer.Close()

	conn, err := net.Dial("tcp", dest.NetAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	const size = 8192
	data := make([]byte, 8192)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	var errg errgroup.Group
	errg.Go(func() error {
		writer := NewWriter(conn)
		mb := MergeBytes(nil, data)

		return writer.WriteMultiBuffer(mb)
	})

	defer func() {
		if err := errg.Wait(); err != nil {
			t.Error(err)
		}
	}()

	rawConn, err := conn.(*net.TCPConn).SyscallConn()
	if err != nil {
		t.Fatal(err)
	}

	reader := NewReadVReader(conn, rawConn, nil)
	var rmb MultiBuffer
	for {
		mb, err := reader.ReadMultiBuffer()
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}
		rmb, _ = MergeMulti(rmb, mb)
		if rmb.Len() == size {
			break
		}
	}

	rdata := make([]byte, size)
	SplitBytes(rmb, rdata)

	if r := cmp.Diff(data, rdata); r != "" {
		t.Fatal(r)
	}
}
