package codec

import (
	"encoding/json"
	"log"
)

//func init() {
//	encoding.RegisterCodec(&JsonFrame{})
//}

type JsonFrame struct {
	RawData json.RawMessage
}

func (f *JsonFrame) Name() string {
	return "json"
}

func (j *JsonFrame) Marshal(v interface{}) ([]byte, error) {
	frame, ok := v.(*JsonFrame)
	if !ok {
		log.Printf("unable to marshal type: %T", v)
		return json.Marshal(v)
	}
	return frame.RawData, nil
}

func (j *JsonFrame) Unmarshal(data []byte, v interface{}) error {
	frame, ok := v.(*JsonFrame)
	if !ok {
		log.Printf("unable to unmarshal type: %T", v)
		return json.Unmarshal(data, v)
	}
	frame.RawData = data
	return nil
}
