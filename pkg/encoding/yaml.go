package encoding

import (
	"bytes"

	"go.yaml.in/yaml/v4"
)

// LoadAndUnmarshalYAML loads data from the specified path and
// decodes it into the specified structure. Unknown fields and
// duplicate keys in the YAML are treated as errors.
func LoadAndUnmarshalYAML(path string, value any) error {
	return LoadAndUnmarshal(path, func(data []byte) error {
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		return decoder.Decode(value)
	})
}
