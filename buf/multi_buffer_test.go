package buf

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMultiBufferRead(t *testing.T) {
	b1 := New()
	if _, err := b1.WriteString("ab"); err != nil {
		t.Fatal(err)
	}

	b2 := New()
	if _, err := b2.WriteString("cd"); err != nil {
		t.Fatal(err)
	}
	mb := MultiBuffer{b1, b2}

	bs := make([]byte, 32)
	_, nBytes := SplitBytes(mb, bs)
	if nBytes != 4 {
		t.Error("expect 4 bytes split, but got ", nBytes)
	}
	if r := cmp.Diff(bs[:nBytes], []byte("abcd")); r != "" {
		t.Error(r)
	}
}

func TestMultiBufferAppend(t *testing.T) {
	var mb MultiBuffer
	b := New()
	if _, err := b.WriteString("ab"); err != nil {
		t.Fatal(err)
	}
	mb = append(mb, b)
	if mb.Len() != 2 {
		t.Error("expected length 2, but got ", mb.Len())
	}
}

func TestMultiBufferSliceBySizeLarge(t *testing.T) {
	lb := make([]byte, 8*1024)
	if _, err := io.ReadFull(rand.Reader, lb); err != nil {
		t.Fatal(err)
	}

	mb := MergeBytes(nil, lb)

	mb, mb2 := SplitSize(mb, 1024)
	if mb2.Len() != 1024 {
		t.Error("expect length 1024, but got ", mb2.Len())
	}
	if mb.Len() != 7*1024 {
		t.Error("expect length 7*1024, but got ", mb.Len())
	}

	mb, mb3 := SplitSize(mb, 7*1024)
	if mb3.Len() != 7*1024 {
		t.Error("expect length 7*1024, but got", mb.Len())
	}

	if !mb.IsEmpty() {
		t.Error("expect empty buffer, but got ", mb.Len())
	}
}

func TestMultiBufferSplitFirst(t *testing.T) {
	b1 := New()
	b1.WriteString("b1")

	b2 := New()
	b2.WriteString("b2")

	b3 := New()
	b3.WriteString("b3")

	var mb MultiBuffer
	mb = append(mb, b1, b2, b3)

	mb, c1 := SplitFirst(mb)
	if diff := cmp.Diff(b1.String(), c1.String()); diff != "" {
		t.Error(diff)
	}

	mb, c2 := SplitFirst(mb)
	if diff := cmp.Diff(b2.String(), c2.String()); diff != "" {
		t.Error(diff)
	}

	mb, c3 := SplitFirst(mb)
	if diff := cmp.Diff(b3.String(), c3.String()); diff != "" {
		t.Error(diff)
	}

	if !mb.IsEmpty() {
		t.Error("expect empty buffer, but got ", mb.String())
	}
}

func TestMultiBufferReadAllToByte(t *testing.T) {
	{
		lb := make([]byte, 8*1024)
		if _, err := io.ReadFull(rand.Reader, lb); err != nil {
			t.Fatal(err)
		}
		rd := bytes.NewBuffer(lb)
		b, err := ReadAllToBytes(rd)
		if err != nil {
			t.Fatal(err)
		}

		if l := len(b); l != 8*1024 {
			t.Error("unexpceted length from ReadAllToBytes", l)
		}
	}
	{
		const dat = "data/test_MultiBufferReadAllToByte.dat"
		f, err := os.Open(dat)
		if err != nil {
			t.Fatal(err)
		}

		buf2, err := ReadAllToBytes(f)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		cnt, err := os.ReadFile(dat)
		if err != nil {
			t.Fatal(err)
		}

		if d := cmp.Diff(buf2, cnt); d != "" {
			t.Error("fail to read from file: ", d)
		}
	}
}

func TestMultiBufferCopy(t *testing.T) {
	lb := make([]byte, 8*1024)
	if _, err := io.ReadFull(rand.Reader, lb); err != nil {
		t.Fatal(err)
	}
	reader := bytes.NewBuffer(lb)

	mb, err := ReadFrom(reader)
	if err != nil {
		t.Fatal(err)
	}

	lbdst := make([]byte, 8*1024)
	mb.Copy(lbdst)

	if d := cmp.Diff(lb, lbdst); d != "" {
		t.Error("unexpceted different from MultiBufferCopy ", d)
	}
}

func TestSplitFirstBytes(t *testing.T) {
	a := New()
	if _, err := a.WriteString("ab"); err != nil {
		t.Fatal(err)
	}
	b := New()
	if _, err := b.WriteString("bc"); err != nil {
		t.Fatal(err)
	}

	mb := MultiBuffer{a, b}

	o := make([]byte, 2)
	_, cnt := SplitFirstBytes(mb, o)
	if cnt != 2 {
		t.Error("unexpected cnt from SplitFirstBytes ", cnt)
	}
	if d := cmp.Diff(string(o), "ab"); d != "" {
		t.Error("unexpected splited result from SplitFirstBytes ", d)
	}
}

func TestCompact(t *testing.T) {
	a := New()
	if _, err := a.WriteString("ab"); err != nil {
		t.Fatal(err)
	}
	b := New()
	if _, err := b.WriteString("bc"); err != nil {
		t.Fatal(err)
	}

	mb := MultiBuffer{a, b}
	cmb := Compact(mb)

	if w := cmb.String(); w != "abbc" {
		t.Error("unexpected Compact result ", w)
	}
}

func BenchmarkSplitBytes(b *testing.B) {
	var mb MultiBuffer
	raw := make([]byte, Size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := StackNew()
		buffer.Extend(Size)
		mb = append(mb, &buffer)
		mb, _ = SplitBytes(mb, raw)
	}
}
