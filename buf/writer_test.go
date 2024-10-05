package buf_test

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/pysugar/wheels/buf"
	"github.com/pysugar/wheels/transport/pipe"
)

func TestWriter(t *testing.T) {
	lb := New()
	if _, err := lb.ReadFrom(rand.Reader); err != nil {
		t.Fatal(err)
	}

	expectedBytes := append([]byte(nil), lb.Bytes()...)

	writeBuffer := bytes.NewBuffer(make([]byte, 0, 1024*1024))

	writer := NewBufferedWriter(NewWriter(writeBuffer))
	writer.SetBuffered(false)

	if err := writer.WriteMultiBuffer(MultiBuffer{lb}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	if r := cmp.Diff(expectedBytes, writeBuffer.Bytes()); r != "" {
		t.Error(r)
	}
}

func TestBytesWriterReadFrom(t *testing.T) {
	const size = 50000
	pReader, pWriter := pipe.New(pipe.WithSizeLimit(size))
	reader := bufio.NewReader(io.LimitReader(rand.Reader, size))
	writer := NewBufferedWriter(pWriter)
	writer.SetBuffered(false)
	nBytes, err := reader.WriteTo(writer)
	if nBytes != size {
		t.Fatal("unexpected size of bytes written: ", nBytes)
	}
	if err != nil {
		t.Fatal("expect success, but actually error: ", err.Error())
	}

	mb, err := pReader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if mb.Len() != size {
		t.Fatal("unexpected size read: ", mb.Len())
	}
}

func TestDiscardBytes(t *testing.T) {
	b := New()
	if _, err := b.ReadFullFrom(rand.Reader, Size); err != nil {
		t.Fatal(err)
	}

	nBytes, err := io.Copy(DiscardBytes, b)
	if err != nil {
		t.Fatal(err)
	}
	if nBytes != Size {
		t.Error("copy size: ", nBytes)
	}
}

func TestDiscardBytesMultiBuffer(t *testing.T) {
	const size = 10240*1024 + 1
	buffer := bytes.NewBuffer(make([]byte, 0, size))
	if _, err := buffer.ReadFrom(io.LimitReader(rand.Reader, size)); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buffer)
	nBytes, err := io.Copy(DiscardBytes, &BufferedReader{Reader: r})
	if err != nil {
		t.Fatal(err)
	}
	if nBytes != size {
		t.Error("copy size: ", nBytes)
	}
}

func TestWriterInterface(t *testing.T) {
	{
		var writer interface{} = (*BufferToBytesWriter)(nil)
		switch writer.(type) {
		case Writer, io.Writer, io.ReaderFrom:
		default:
			t.Error("BufferToBytesWriter is not Writer, io.Writer or io.ReaderFrom")
		}
	}

	{
		var writer interface{} = (*BufferedWriter)(nil)
		switch writer.(type) {
		case Writer, io.Writer, io.ReaderFrom, io.ByteWriter:
		default:
			t.Error("BufferedWriter is not Writer, io.Writer, io.ReaderFrom or io.ByteWriter")
		}
	}
}
