package encoding

import (
	"github.com/BurntSushi/toml"
)

func LoadAndUnmarshalTOML(path string, value interface{}) error {
	return loadAndUnmarshal(path, func(data []byte) error {
		return toml.Unmarshal(data, value)
	})
}
