package encoding

import (
	"github.com/BurntSushi/toml"
)

// LoadAndUnmarshalTOML loads data from the specified path and decodes it into
// the specified structure.
func LoadAndUnmarshalTOML(path string, value interface{}) error {
	return LoadAndUnmarshal(path, func(data []byte) error {
		return toml.Unmarshal(data, value)
	})
}
