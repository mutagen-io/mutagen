package encoding

import (
	"github.com/gogo/protobuf/proto"
)

func LoadAndUnmarshalProtobuf(path string, message proto.Message) error {
	return loadAndUnmarshal(path, func(data []byte) error {
		return proto.Unmarshal(data, message)
	})
}

func MarshalAndSaveProtobuf(path string, message proto.Message) error {
	return marshalAndSave(path, func() ([]byte, error) {
		return proto.Marshal(message)
	})
}
