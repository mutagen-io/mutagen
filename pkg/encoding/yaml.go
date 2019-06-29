package encoding

import (
	"gopkg.in/yaml.v2"
)

// LoadAndUnmarshalYAML loads data from the specified path and decodes it into
// the specified structure.
func LoadAndUnmarshalYAML(path string, value interface{}) error {
	return LoadAndUnmarshal(path, func(data []byte) error {
		return yaml.UnmarshalStrict(data, value)
	})
}
