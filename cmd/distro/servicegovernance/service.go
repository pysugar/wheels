package servicegovernance

import (
	"encoding/json"
	"fmt"
)

const (
	DefaultGroup = "default"
)

type (
	Endpoint struct {
		Address  string              `json:"address"`
		Group    string              `json:"group"`
		Metadata map[string][]string `json:"metadata"`
	}

	Instance struct {
		Env         string
		ServiceName string
		Endpoint    Endpoint
	}
)

func (i *Instance) ServiceWithEnv() string {
	return fmt.Sprintf("/%s/%s", i.Env, i.ServiceName)
}

func (i *Instance) Key() string {
	serviceWithEnv := i.ServiceWithEnv()
	serviceKeyPrefix := fmt.Sprintf("%s/", serviceWithEnv)
	if i.Endpoint.Group != DefaultGroup {
		serviceKeyPrefix = fmt.Sprintf("%s:%s/", serviceWithEnv, i.Endpoint.Group)
	}
	return fmt.Sprintf("%s%s", serviceKeyPrefix, i.Endpoint.Address)
}

func (e *Endpoint) Encode() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e *Endpoint) Decode(value []byte) error {
	return json.Unmarshal(value, e)
}
