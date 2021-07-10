package redis

import (
	"encoding/json"
)

type message struct {
	ToAll  bool
	Room   string
	Except []string
	Data   interface{}
}

func (m *message) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m *message) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}
