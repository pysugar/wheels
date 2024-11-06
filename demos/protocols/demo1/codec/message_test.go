package codec_test

import (
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
	"testing"
)

func TestMessage_EncodeDecode(t *testing.T) {
	authPayload := []byte("user:password")
	authMsg := &codec.Message{
		Version: 1,
		Type:    codec.MSG_TYPE_AUTH,
		MsgID:   12345,
		Payload: authPayload,
	}

	// Encode the message
	encodedAuth, err := authMsg.Encode()
	if err != nil {
		t.Errorf("Encode Error: %v", err)
		return
	}
	t.Logf("Encoded Message: %v", encodedAuth)

	// Decode the message
	decodedMsg, err := codec.Decode(encodedAuth)
	if err != nil {
		t.Errorf("Decode Error: %v", err)
		return
	}
	t.Logf("Decoded Message:\n Version: %d\n Type: %d\n Length: %d\n MsgID: %d\n Payload: %s\n Checksum: %d\n",
		decodedMsg.Version, decodedMsg.Type, decodedMsg.Length, decodedMsg.MsgID, decodedMsg.Payload, decodedMsg.Checksum)

}
