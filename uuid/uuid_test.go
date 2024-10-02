package uuid_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/pysugar/wheels/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseBytes(t *testing.T) {
	str := "2418d087-648d-4990-86e8-19dca1d006d3"
	bytes := []byte{0x24, 0x18, 0xd0, 0x87, 0x64, 0x8d, 0x49, 0x90, 0x86, 0xe8, 0x19, 0xdc, 0xa1, 0xd0, 0x06, 0xd3}

	id, err := uuid.ParseBytes(bytes)
	assert.NoError(t, err)

	if diff := cmp.Diff(id.String(), str); diff != "" {
		t.Error(diff)
	}

	_, err = uuid.ParseBytes([]byte{1, 3, 2, 4})
	if err == nil {
		t.Fatal("Expect error but nil")
	}
}

func TestParseString(t *testing.T) {
	str := "2418d087-648d-4990-86e8-19dca1d006d3"
	expectedBytes := []byte{0x24, 0x18, 0xd0, 0x87, 0x64, 0x8d, 0x49, 0x90, 0x86, 0xe8, 0x19, 0xdc, 0xa1, 0xd0, 0x06, 0xd3}

	id, err := uuid.ParseString(str)
	assert.NoError(t, err)

	if r := cmp.Diff(expectedBytes, id.Bytes()); r != "" {
		t.Fatal(r)
	}

	u0, _ := uuid.ParseString("example")
	u5, _ := uuid.ParseString("feb54431-301b-52bb-a6dd-e1e93e81bb9e")
	if r := cmp.Diff(u0, u5); r != "" {
		t.Fatal(r)
	}

	_, err = uuid.ParseString("2418d087-648k-4990-86e8-19dca1d006d3")
	if err == nil {
		t.Fatal("Expect error but nil")
	}
}

func TestNewUUID(t *testing.T) {
	id := uuid.New()
	id2, err := uuid.ParseString(id.String())
	assert.NoError(t, err)

	if id.String() != id2.String() {
		t.Error("uuid string: ", id.String(), " != ", id2.String())
	}
	if r := cmp.Diff(id.Bytes(), id2.Bytes()); r != "" {
		t.Error(r)
	}
}

func TestRandom(t *testing.T) {
	id := uuid.New()
	id2 := uuid.New()

	if id.String() == id2.String() {
		t.Error("duplicated uuid")
	}
}

func TestEquals(t *testing.T) {
	var id *uuid.UUID
	var id2 *uuid.UUID
	if !id.Equals(id2) {
		t.Error("empty uuid should equal")
	}

	id3 := uuid.New()
	if id.Equals(&id3) {
		t.Error("nil uuid equals non-nil uuid")
	}
}
