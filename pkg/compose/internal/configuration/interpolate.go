package configuration

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/compose-spec/compose-go/template"
)

// interpolate performs recursive interpolation on a raw YAML hierarchy using
// the specified mapping. It only performs interpolation on scalar value nodes, not keys.
func interpolate(node *yaml.Node, mapping template.Mapping) error {
	// Handle interpolation based on the node type.
	switch node.Kind {
	case yaml.DocumentNode:
		// Somewhat counterintuitively, document nodes aren't structured like
		// mapping nodes. Instead, they're basically sequence nodes containing
		// either no content nodes (in the case of an empty document) or a
		// single mapping content node containing the root document content.
		// This is why we fall through to the sequence node handling as opposed
		// to the mapping node handling.
		fallthrough
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := interpolate(child, mapping); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return errors.New("mapping node with unbalanced key/value count")
		}
		for i := 1; i < len(node.Content); i += 2 {
			if err := interpolate(node.Content[i], mapping); err != nil {
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
