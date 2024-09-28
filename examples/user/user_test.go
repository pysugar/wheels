package user_test

import (
	"fmt"
	"github.com/pysugar/wheels/examples/user"
	"github.com/pysugar/wheels/serial"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"reflect"
	"testing"
)

func init() {
	protoregistry.GlobalTypes.RangeExtensions(func(et protoreflect.ExtensionType) bool {
		fmt.Println(et)
		return true
	})
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		fmt.Println(mt.Descriptor().FullName())
		return true
	})
}

func TestEncode(t *testing.T) {
	acct := &user.Account{Username: "gosuger", Password: "xxxxxx"}
	t.Log(acct, reflect.TypeOf(acct))

	msg := serial.Encode(acct)
	t.Log(msg)

	acct2, _ := serial.Decode(msg)
	t.Log(acct2, reflect.TypeOf(acct2))

	t.Log(acct2.(*user.Account).Username)
}
