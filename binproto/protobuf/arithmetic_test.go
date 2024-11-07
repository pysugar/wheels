package protobuf

import (
	"fmt"
	"testing"
)

func TestIntRepresentation(t *testing.T) {
	var a int32 = -1
	t.Logf("a = %032b", a)
	var b = uint32(a)
	t.Logf("b = %032b", b)
	var c int32 = int32(b)
	t.Logf("c = %032b", c)

	printInt32Binary(-128)
	printInt64Binary(-127)
}

func TestRightShift(t *testing.T) {
	var a int32 = -128 // 二进制: 11111111 11111111 11111111 10000000
	var b = uint32(a)  // 二进制: 11111111 11111111 11111111 10000000

	fmt.Printf("(binary: %032b) %d\n", uint32(a), a)
	fmt.Printf("(binary: %032b) %d\n", b, b)

	aRightShift := a >> 2
	bRightShift := b >> 2

	fmt.Printf("int32 arithmetic right shift 2:\t (binary: %032b) %d\n", uint32(aRightShift), aRightShift)
	fmt.Printf("uint32 logical right shift 2:\t (binary: %032b) %d\n", bRightShift, bRightShift)

	printInt32Binary(aRightShift)
	t.Logf("bRightShift (%d) = %032b", bRightShift, bRightShift)

	c := arithmeticRightShiftInt32(a, 3)
	fmt.Printf("int32 arithmetic right shift 3:\t (binary: %032b) %d\n", uint32(c), c)
	d := logicalRightShiftInt32(a, 3)
	fmt.Printf("int32 logical right shift 3:\t (binary: %032b) %d\n", d, d)
}

func TestZigZag(t *testing.T) {
	for i := int32(0); i < 10; i++ {
		t.Logf("[encode] origin: %d, zigzap: %d", i, ZigZagEncode32(i))
	}
	for i := int32(-10); i < 0; i++ {
		t.Logf("[encode]origin: %d, zigzap: %d", i, ZigZagEncode32(i))
	}

	for i := uint32(0); i < 20; i += 2 {
		t.Logf("[decode] origin: %d, zigzap: %d", i, ZigZagDecode32(i))
	}
	for i := uint32(1); i < 20; i += 2 {
		t.Logf("[decode]origin: %d, zigzap: %d", i, ZigZagDecode32(i))
	}
}

func printInt32Binary(n int32) {
	fmt.Printf("int32: %d\n", n)
	fmt.Printf("Binary: %032b\n", uint32(n))
}

func printInt64Binary(n int64) {
	fmt.Printf("int64: %d\n", n)
	fmt.Printf("Binary: %064b\n", uint64(n))
}

func logicalRightShiftInt32(n int32, shift uint) int32 {
	return int32(uint32(n) >> shift)
}

func arithmeticRightShiftInt32(n int32, shift uint) int32 {
	return n >> shift
}

func ZigZagEncode32(n int32) uint32 {
	return uint32((n << 1) ^ (n >> 31))
}

func ZigZagDecode32(n uint32) int32 {
	return int32((n >> 1) ^ uint32(-(n & 1)))
}
