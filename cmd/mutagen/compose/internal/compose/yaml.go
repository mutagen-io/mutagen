package compose

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/compose-spec/compose-go/template"

	"github.com/mutagen-io/mutagen/pkg/encoding"
)

// NewEmptyMapping creates an interpolation mapping with no entries.
func NewEmptyMapping() template.Mapping {
	return func(_ string) (string, bool) {
		return "", false
	}
}

// NewMapMapping converts a map[string]string to an interpolation mapping.
func NewMapMapping(mapping map[string]string) template.Mapping {
	return func(key string) (string, bool) {
		value, ok := mapping[key]
		return value, ok
	}
}

// interpolateNode performs recursive interpolation on a raw YAML hierarchy
// using the specified mapping. It only performs interpolation on scalar value
// nodes.
func interpolateNode(node *yaml.Node, mapping template.Mapping) error {
	// Handle interpolation based on the node type.
	switch node.Kind {
	case yaml.DocumentNode:
		fallthrough
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := interpolateNode(child, mapping); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return errors.New("mapping node with unbalanced key/value count")
		}
		for i := 1; i < len(node.Content); i += 2 {
			if err := interpolateNode(node.Content[i], mapping); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		var err error
		if node.Value, err = template.Substitute(node.Value, mapping); err != nil {
			return fmt.Errorf("unable to interpolate value (%s): %w", node.Value, err)
		}
	case yaml.AliasNode:
	default:
		return errors.New("unknown YAML node kind")
	}

	// Success.
	return nil
}

// UnmarshalAndInterpolateYAML unmarshals YAML data, performs interpolation
// (before type conversion), and then decodes the result into the specified
// value. If mapping is nil, then no interpolation is performed (i.e. ${...}
// expressions will be left in-place. To perform interpolation without providing
// any variables, use NewEmptyMapping. Unlike the standard LoadAndUnmarshalYAML
// function, this function does not support strict decoding (i.e. it allows
// unknown keys to pass silently).
// TODO: Enable strict decoding once go-yaml/yaml#460 is resolved (potentially
// by go-yaml/yaml#557). Actually, it may be advantageous to us to continue
// allowing non-strict decoding so that we can support Docker Compose files with
// other top-level extension fields, but it would be nice if we could at least
// use strict decoding for the Mutagen configuration portion.
func UnmarshalAndInterpolateYAML(data []byte, mapping template.Mapping, value interface{}) error {
	// Perform a generic parsing operation on the data.
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	// If a mapping has been specified, then perform interpolation.
	if mapping != nil {
		if err := interpolateNode(&root, mapping); err != nil {
			return fmt.Errorf("interpolation failed: %w", err)
		}
	}

	// Perform decoding.
	return root.Decode(value)
}

// LoadAndUnmarshalAndInterpolateYAML loads YAML data from a file and then
// processes it using UnmarshalAndInterpolateYAML.
func LoadAndUnmarshalAndInterpolateYAML(path string, mapping template.Mapping, value interface{}) error {
	return encoding.LoadAndUnmarshal(path, func(data []byte) error {
		return UnmarshalAndInterpolateYAML(data, mapping, value)
	})
}
