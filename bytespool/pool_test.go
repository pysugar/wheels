package bytespool_test

import (
	"github.com/pysugar/wheels/bytespool"
	"testing"
)

func TestAllocFree(t *testing.T) {
	for i := int32(1); i <= 20; i++ {
		size := i * 1024 * i
		bytes := bytespool.Alloc(size)
		t.Logf("%d:\t[%d]byte@%p", size, len(bytes), bytes)
		bytespool.Free(bytes)
	}
}
