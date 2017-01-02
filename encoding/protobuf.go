package encoding

import (
	"github.com/golang/protobuf/proto"
)

// TODO: Now that this package no longer aims to support JSON encoding, it might
// make sense to flatten these functions and relocate them to the session
// package, which is the only place that they're used. But I can see us using
// them in other packages, so I'll let this idea simmer, and we might even bring
// JSON back for other purposes.

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
