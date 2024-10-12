package mocks_test

import (
	"github.com/golang/mock/gomock"
	"github.com/pysugar/wheels/testing/mocks"
	"io"
	"testing"
)

func ProcessData(r io.Reader) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

func TestProcessData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := mocks.NewReader(ctrl)
	mockReader.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(p []byte) (int, error) {
			n := copy(p, []byte("hello"))
			t.Logf("doAndReturn: %d, %d", len(p), n)
			return n, nil
		})
	//mockReader.
	//	EXPECT().
	//	Read(gomock.Any()).
	//	Do(func(p []byte) {
	//		n := copy(p, []byte("hello"))
	//		t.Logf("do: %d, %d", len(p), n)
	//	}).
	//	Return(5, nil)

	result, err := ProcessData(mockReader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []byte("hello")
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
