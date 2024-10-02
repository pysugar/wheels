package units_test

import (
	"github.com/pysugar/wheels/units"
	"testing"
)

func TestByteSizes(t *testing.T) {
	assertSizeString(t, units.ByteSize(0), "0")

	tests := []struct {
		name     string
		size     units.ByteSize
		expected string
	}{
		{"Bytes", 1, "1.00B"},
		{"Kilobytes", 1 << 10, "1.00KB"},
		{"Megabytes", 1 << 20, "1.00MB"},
		{"Gigabytes", 1 << 30, "1.00GB"},
		{"Terabytes", 1 << 40, "1.00TB"},
		{"Petabytes", 1 << 50, "1.00PB"},
		{"Exabytes", 1 << 60, "1.00EB"},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			assertSizeValue(t, assertSizeString(t, tt.size, tt.expected), tt.size)
		})
	}
}

func assertSizeValue(t *testing.T, size string, expected units.ByteSize) {
	actual := units.ByteSize(0)
	err := actual.Parse(size)
	if err != nil {
		t.Error(err)
	}
	if actual != expected {
		t.Errorf("expect %s, but got %s", expected, actual)
	}
}

func assertSizeString(t *testing.T, size units.ByteSize, expected string) string {
	actual := size.String()
	if actual != expected {
		t.Errorf("expect %s, but got %s", expected, actual)
	}
	return expected
}
