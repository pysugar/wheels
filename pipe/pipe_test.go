package pipe_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pysugar/wheels/buf"
	"github.com/pysugar/wheels/lang"
	. "github.com/pysugar/wheels/pipe"
	"golang.org/x/sync/errgroup"
)

func TestPipeReadWrite(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))

	b := buf.New()
	b.WriteString("abcd")
	if err := pWriter.WriteMultiBuffer(buf.MultiBuffer{b}); err != nil {
		t.Fatal(err)
	}

	b2 := buf.New()
	b2.WriteString("efg")
	if err := pWriter.WriteMultiBuffer(buf.MultiBuffer{b2}); err != nil {
		t.Fatal(err)
	}

	rb, err := pReader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if r := cmp.Diff(rb.String(), "abcdefg"); r != "" {
		t.Error(r)
	}
}

func TestPipeInterrupt(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))
	payload := []byte{'a', 'b', 'c', 'd'}
	b := buf.New()
	b.Write(payload)
	if err := pWriter.WriteMultiBuffer(buf.MultiBuffer{b}); err != nil {
		t.Fatal(err)
	}
	pWriter.Interrupt()

	rb, err := pReader.ReadMultiBuffer()
	if err != io.ErrClosedPipe {
		t.Fatal("expect io.ErrClosePipe, but got ", err)
	}
	if !rb.IsEmpty() {
		t.Fatal("expect empty buffer, but got ", rb.Len())
	}
}

func TestPipeClose(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))
	payload := []byte{'a', 'b', 'c', 'd'}
	b := buf.New()
	if _, err := b.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := pWriter.WriteMultiBuffer(buf.MultiBuffer{b}); err != nil {
		t.Fatal(err)
	}
	if err := pWriter.Close(); err != nil {
		t.Fatal(err)
	}

	rb, err := pReader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if rb.String() != string(payload) {
		t.Fatal("expect content ", string(payload), " but actually ", rb.String())
	}

	rb, err = pReader.ReadMultiBuffer()
	if err != io.EOF {
		t.Fatal("expected EOF, but got ", err)
	}
	if !rb.IsEmpty() {
		t.Fatal("expect empty buffer, but got ", rb.String())
	}
}

func TestPipeLimitZero(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(0))
	bb := buf.New()
	if _, err := bb.Write([]byte{'a', 'b'}); err != nil {
		t.Fatal(err)
	}
	if err := pWriter.WriteMultiBuffer(buf.MultiBuffer{bb}); err != nil {
		t.Fatal(err)
	}

	var errg errgroup.Group
	errg.Go(func() error {
		b := buf.New()
		b.Write([]byte{'c', 'd'})
		return pWriter.WriteMultiBuffer(buf.MultiBuffer{b})
	})
	errg.Go(func() error {
		time.Sleep(time.Second)

		var container buf.MultiBufferContainer
		if err := buf.Copy(pReader, &container); err != nil {
			return err
		}

		if r := cmp.Diff(container.String(), "abcd"); r != "" {
			return errors.New(r)
		}
		return nil
	})
	errg.Go(func() error {
		time.Sleep(time.Second * 2)
		return pWriter.Close()
	})
	if err := errg.Wait(); err != nil {
		t.Error(err)
	}
}

func TestPipeWriteMultiThread(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(0))

	var errg errgroup.Group
	for i := 0; i < 10; i++ {
		errg.Go(func() error {
			b := buf.New()
			b.WriteString("abcd")
			return pWriter.WriteMultiBuffer(buf.MultiBuffer{b})
		})
	}
	time.Sleep(time.Millisecond * 100)
	pWriter.Close()
	errg.Wait()

	b, err := pReader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if r := cmp.Diff(b[0].Bytes(), []byte{'a', 'b', 'c', 'd'}); r != "" {
		t.Error(r)
	}
}

func TestInterfaces(t *testing.T) {
	_ = (buf.Reader)(new(Reader))
	_ = (buf.TimeoutReader)(new(Reader))

	_ = (lang.Interruptible)(new(Reader))
	_ = (lang.Interruptible)(new(Writer))
	_ = (lang.Closable)(new(Writer))
}

func BenchmarkPipeReadWrite(b *testing.B) {
	reader, writer := New(WithoutSizeLimit())
	a := buf.New()
	a.Extend(buf.Size)
	c := buf.MultiBuffer{a}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := writer.WriteMultiBuffer(c); err != nil {
			b.Fatal(err)
		}
		d, err := reader.ReadMultiBuffer()
		if err != nil {
			b.Fatal(err)
		}
		c = d
	}
}
