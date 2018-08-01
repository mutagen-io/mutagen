package encoding

import (
	"github.com/golang/protobuf/proto"
)

// LoadAndUnmarshalProtobuf loads data from the specified path and decodes it
// into the specified Protocol Buffers message.
func LoadAndUnmarshalProtobuf(path string, message proto.Message) error {
	return loadAndUnmarshal(path, func(data []byte) error {
		return proto.Unmarshal(data, message)
	})
}

// MarshalAndSaveProtobuf marshals the specified Protocol Buffers message and
// saves it to the specified path.
func MarshalAndSaveProtobuf(path string, message proto.Message) error {
	return marshalAndSave(path, func() ([]byte, error) {
		return proto.Marshal(message)
	})
}
