package codec

import (
	"encoding/json"
)

func init() {
	// encoding.RegisterCodec(jsonCodec{})
}

type JsonFrame struct {
	RawData []byte
}

func (f *JsonFrame) Name() string {
	return "json"
}

func (f *JsonFrame) Marshal(v interface{}) ([]byte, error) {
	frame, _ := v.(*JsonFrame)
	return frame.RawData, nil
}

func (f *JsonFrame) Unmarshal(data []byte, v interface{}) error {
	frame, _ := v.(*JsonFrame)
	frame.RawData = data
	return nil
}

type jsonCodec struct{}

func (jc jsonCodec) Name() string {
	return "json"
}

func (jc jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jc jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jc jsonCodec) String() string {
	return jc.Name()
}
