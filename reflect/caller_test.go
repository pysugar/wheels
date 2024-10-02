package reflect_test

import (
	"runtime"
	"testing"
)

func TestCaller(t *testing.T) {
	pc, file, line, ok := runtime.Caller(1)
	if ok {
		funcInfo := runtime.FuncForPC(pc)
		funcName := funcInfo.Name()
		t.Logf("[main] Called from function: %s\nFile: %s\nLine: %d\n", funcName, file, line)
	}

	func1(t)
}

func func1(t *testing.T) {
	pc, file, line, ok := runtime.Caller(1)
	if ok {
		funcInfo := runtime.FuncForPC(pc)
		funcName := funcInfo.Name()
		t.Logf("[func1] Called from function: %s\nFile: %s\nLine: %d\n", funcName, file, line)
	}

	func2(t)
}

func func2(t *testing.T) {
	pc, file, line, ok := runtime.Caller(1)
	if ok {
		funcInfo := runtime.FuncForPC(pc)
		funcName := funcInfo.Name()
		t.Logf("[func2] Called from function: %s\nFile: %s\nLine: %d\n", funcName, file, line)
	}

	func3(t)
}

func func3(t *testing.T) {
	pc, file, line, ok := runtime.Caller(1)
	if ok {
		funcInfo := runtime.FuncForPC(pc)
		funcName := funcInfo.Name()
		t.Logf("[func3] Called from function: %s\nFile: %s\nLine: %d\n", funcName, file, line)
	}
}
