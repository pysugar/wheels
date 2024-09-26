package user

import (
	"github.com/pysugar/wheels/serial"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	acct := &Account{Username: "gosuger", Password: "xxxxxx"}
	t.Log(acct, reflect.TypeOf(acct))

	msg := serial.Encode(acct)
	t.Log(msg)

	acct2, _ := serial.Decode(msg)
	t.Log(acct2, reflect.TypeOf(acct2))

	t.Log(acct2.(*Account).Username)
}
