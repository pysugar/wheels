package buf_test

import (
	"bytes"
	"crypto/rand"
	"github.com/google/go-cmp/cmp"
	. "github.com/pysugar/wheels/buf"
	"testing"
)

func TestBufferClear(t *testing.T) {
	buffer := New()
	defer buffer.Release()

	payload := "Bytes"
	buffer.Write([]byte(payload))
	if diff := cmp.Diff(buffer.Bytes(), []byte(payload)); diff != "" {
		t.Error(diff)
	}

	buffer.Clear()
	if buffer.Len() != 0 {
		t.Error("expect 0 length, but got ", buffer.Len())
	}
}

func TestBufferIsEmpty(t *testing.T) {
	buffer := New()
	defer buffer.Release()

	if buffer.IsEmpty() != true {
		t.Error("expect empty buffer, but not")
	}
}

func TestBufferString(t *testing.T) {
	buffer := New()
	defer buffer.Release()

	const payload = "Test String"
	_, err := buffer.WriteString(payload)
	if err != nil {
		t.Fatal(err)
	}

	if buffer.String() != payload {
		t.Error("expect buffer content as ", payload, " but actually ", buffer.String())
	}
}

func TestBufferByte(t *testing.T) {
	{
		buffer := New()
		err := buffer.WriteByte('m')
		if err != nil {
			t.Fatal(err)
		}
		if buffer.String() != "m" {
			t.Error("expect buffer content as ", "m", " but actually ", buffer.String())
		}
		buffer.Release()
	}
	{
		buffer := StackNew()
		err := buffer.WriteByte('n')
		if err != nil {
			t.Fatal(err)
		}
		if buffer.String() != "n" {
			t.Error("expect buffer content as ", "n", " but actually ", buffer.String())
		}
		buffer.Release()
	}
	{
		buffer := StackNew()
		_, err := buffer.WriteString("HELLOWORLD")
		if err != nil {
			t.Fatal(err)
		}
		if b := buffer.Byte(5); b != 'W' {
			t.Error("unexpected byte ", b)
		}

		buffer.SetByte(5, 'M')
		if buffer.String() != "HELLOMORLD" {
			t.Error("expect buffer content as ", "n", " but actually ", buffer.String())
		}
		buffer.Release()
	}
}

func TestBufferResize(t *testing.T) {
	buffer := New()
	defer buffer.Release()

	const payload = "Test String"
	_, err := buffer.WriteString(payload)
	if err != nil {
		t.Fatal(err)
	}
	if buffer.String() != payload {
		t.Error("expect buffer content as ", payload, " but actually ", buffer.String())
	}

	buffer.Resize(-6, -3)
	if l := buffer.Len(); int(l) != 3 {
		t.Error("len error ", l)
	}

	if s := buffer.String(); s != "Str" {
		t.Error("unexpect buffer ", s)
	}

	buffer.Resize(int32(len(payload)), 200)
	if l := buffer.Len(); int(l) != 200-len(payload) {
		t.Error("len error ", l)
	}
}

func TestBufferSlice(t *testing.T) {
	{
		b := New()
		_, err := b.Write([]byte("abcd"))
		if err != nil {
			t.Fatal(err)
		}
		bytes := b.BytesFrom(-2)
		if diff := cmp.Diff(bytes, []byte{'c', 'd'}); diff != "" {
			t.Error(diff)
		}
	}

	{
		b := New()
		_, err := b.Write([]byte("abcd"))
		if err != nil {
			t.Fatal(err)
		}
		bytes := b.BytesTo(-2)
		if diff := cmp.Diff(bytes, []byte{'a', 'b'}); diff != "" {
			t.Error(diff)
		}
	}

	{
		b := New()
		_, err := b.Write([]byte("abcd"))
		if err != nil {
			t.Fatal(err)
		}
		bytes := b.BytesRange(-3, -1)
		if diff := cmp.Diff(bytes, []byte{'b', 'c'}); diff != "" {
			t.Error(diff)
		}
	}
}

func TestBufferReadFullFrom(t *testing.T) {
	payload := make([]byte, 1024)
	_, err := rand.Read(payload)
	if err != nil {
		t.Fatal(err)
	}

	reader := bytes.NewReader(payload)
	b := New()
	n, err := b.ReadFullFrom(reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024 {
		t.Error("expect reading 1024 bytes, but actually ", n)
	}

	if diff := cmp.Diff(payload, b.Bytes()); diff != "" {
		t.Error(diff)
	}
}

func BenchmarkNewBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buffer := New()
		buffer.Release()
	}
}

func BenchmarkNewBufferStack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buffer := StackNew()
		buffer.Release()
	}
}

func BenchmarkWrite2(b *testing.B) {
	buffer := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buffer.Write([]byte{'a', 'b'})
		buffer.Clear()
	}
}

func BenchmarkWrite8(b *testing.B) {
	buffer := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buffer.Write([]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'})
		buffer.Clear()
	}
}

func BenchmarkWrite32(b *testing.B) {
	buffer := New()
	payload := make([]byte, 32)
	rand.Read(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buffer.Write(payload)
		buffer.Clear()
	}
}

func BenchmarkWriteByte2(b *testing.B) {
	buffer := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.WriteByte('a')
		_ = buffer.WriteByte('b')
		buffer.Clear()
	}
}

func BenchmarkWriteByte8(b *testing.B) {
	buffer := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.WriteByte('a')
		_ = buffer.WriteByte('b')
		_ = buffer.WriteByte('c')
		_ = buffer.WriteByte('d')
		_ = buffer.WriteByte('e')
		_ = buffer.WriteByte('f')
		_ = buffer.WriteByte('g')
		_ = buffer.WriteByte('h')
		buffer.Clear()
	}
}
