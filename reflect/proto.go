package reflect

import (
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func IsLegacyProtoMessage(msg any) bool {
	if msg == nil {
		return false
	}
	_, isNew := msg.(protoadapt.MessageV2)
	_, isLegacy := msg.(protoadapt.MessageV1)
	return isLegacy && !isNew
}

func IsProtoMessage(msg any) bool {
	if msg == nil {
		return false
	}

	//_, ok := msg.(proto.Message)
	//return ok
	if m, ok := msg.(interface{ ProtoReflect() protoreflect.Message }); ok {
		return m.ProtoReflect() != nil
	}
	return false
}
