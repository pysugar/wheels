package serial

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func Encode(message proto.Message) *TypedMessage {
	if message == nil {
		return nil
	}
	value, _ := proto.Marshal(message)
	return &TypedMessage{
		Type:  GetMessageType(message),
		Value: value,
	}
}

func Decode(v *TypedMessage) (proto.Message, error) {
	if v == nil {
		return nil, errors.New("nil")
	}

	protoMessage, err := GetInstance(v.Type)
	if err != nil {
		return nil, err
	}
	if err := proto.Unmarshal(v.Value, protoMessage); err != nil {
		return nil, err
	}
	return protoMessage, nil
}

func GetInstance(messageType string) (proto.Message, error) {
	messageTypeDescriptor := protoreflect.FullName(messageType)
	mType, err := protoregistry.GlobalTypes.FindMessageByName(messageTypeDescriptor)
	if err != nil {
		return nil, err
	}
	return mType.New().Interface(), nil
}

func GetMessageType(message proto.Message) string {
	return string(message.ProtoReflect().Descriptor().FullName())
}
