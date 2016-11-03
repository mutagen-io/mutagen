package encoding

import (
	"encoding/json"
)

func LoadAndUnmarshalJSON(path string, message interface{}) error {
	return loadAndUnmarshal(path, func(data []byte) error {
		return json.Unmarshal(data, message)
	})
}

func MarshalAndSaveJSON(path string, message interface{}) error {
	return marshalAndSave(path, func() ([]byte, error) {
		return json.Marshal(message)
	})
}
